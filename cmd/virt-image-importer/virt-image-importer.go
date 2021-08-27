package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type client struct {
	ec2Client *ec2.Client
	stsClient *sts.Client
	s3Client  *s3.Client
	region    string
}

const (
	ExportImageFormat = "vmdk"
)

func NewClient(region string) (*client, error) {

	// Load the SDK's configuration from environment and shared config, and
	// create the ec2Client with this.
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	if region == "" {
		region = cfg.Region
	}

	ec2Client := ec2.NewFromConfig(cfg)
	stsClient := sts.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

	return &client{
		ec2Client: ec2Client,
		stsClient: stsClient,
		s3Client:  s3Client,
		region:    region,
	}, nil
}

func (c *client) FindGlobalImageById(amiId string) (*types.Image, error) {

	params := &ec2.DescribeImagesInput{
		ImageIds: []string{amiId},
	}

	amiListOutput, err := c.ec2Client.DescribeImages(context.Background(), params, func(o *ec2.Options) {
		o.Region = c.region
	})
	if err != nil {
		return nil, err
	}

	if len(amiListOutput.Images) == 0 {
		return nil, fmt.Errorf("image with id %s not found", amiId)
	}

	image := amiListOutput.Images[0]

	return &image, nil
}

func (c *client) ExportImage(amiId string, s3Bucket string, s3Prefix string, imageFormat string) (string, error) {

	if imageFormat != "vmdk" {
		return "", fmt.Errorf("Image format %s is unsupported, only image format 'vmdk' is supported", imageFormat)
	}

	description := fmt.Sprintf("Exporting ami %s for import into KubeVirt cluster", amiId)
	params := &ec2.ExportImageInput{
		DiskImageFormat: types.DiskImageFormatVmdk,
		ImageId:         &amiId,
		S3ExportLocation: &types.ExportTaskS3LocationRequest{
			S3Bucket: &s3Bucket,
			S3Prefix: &s3Prefix,
		},
		Description: &description,
	}

	amiExportOutput, err := c.ec2Client.ExportImage(context.Background(), params, func(o *ec2.Options) {
		o.Region = c.region
	})

	if err != nil {
		return "", err
	}

	if amiExportOutput.ExportImageTaskId == nil {
		return "", fmt.Errorf("No export task id provided for ExportImage job. Unable to track export")
	}

	return *amiExportOutput.ExportImageTaskId, nil
}

func (c *client) GetExportTaskStatus(exportTaskId string) (string, error) {
	params := &ec2.DescribeExportImageTasksInput{
		ExportImageTaskIds: []string{exportTaskId},
	}

	exportTaskOutput, err := c.ec2Client.DescribeExportImageTasks(context.Background(), params, func(o *ec2.Options) {
		o.Region = c.region
	})
	if err != nil {
		return "", err
	}

	if len(exportTaskOutput.ExportImageTasks) == 0 || exportTaskOutput.ExportImageTasks[0].Status == nil {
		return "", fmt.Errorf("Export task id %d not found. Unable to track export", exportTaskId)
	}

	return *exportTaskOutput.ExportImageTasks[0].Status, nil
}

func (c *client) WaitForExportImageCompletion(taskId string, timeout time.Duration) error {
	ticker := time.NewTicker(timeout).C
	pollTicker := time.NewTicker(time.Second * 15).C

	log.Printf("Polling task id %s to determine if it is completed", taskId)
	status, _ := c.GetExportTaskStatus(taskId)
	if status == "completed" {
		return nil
	}

	// if not available, poll until available or timeout is hit
	for {
		select {
		case <-ticker:
			return fmt.Errorf("timed out waiting for task id %s to become complete", taskId)
		case <-pollTicker:
			log.Printf("Polling task id %s to determine if it is completed", taskId)

			status, err := c.GetExportTaskStatus(taskId)
			if err != nil {
				log.Printf("err encountered looking up task id %s: %v", taskId, err)
				continue
			} else if status == "completed" {
				log.Printf("Task id %s completed", taskId)
				return nil
			}
		}
	}
}

func (c *client) FindImageByName(amiName string, accountId string) (*types.Image, bool, error) {
	filterKeyName := "name"
	params := &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   &filterKeyName,
				Values: []string{amiName},
			},
		},
		Owners: []string{accountId},
	}

	amiListOutput, err := c.ec2Client.DescribeImages(context.Background(), params, func(o *ec2.Options) {
		o.Region = c.region
	})
	if err != nil {
		return nil, false, err
	}

	if len(amiListOutput.Images) == 0 {
		return nil, false, nil
	}

	image := amiListOutput.Images[0]
	return &image, true, nil
}

func (c *client) CopyImageName(amiId string) string {
	return fmt.Sprintf("kubevirt-export-automation-copy-%s", amiId)
}

func (c *client) ExpectedImageS3Key(amiId string, imageFormat string) string {
	return fmt.Sprintf("export-%s.%s", amiId, imageFormat)
}

func (c *client) CopyImage(amiId string, amiCopyName string) (string, error) {
	copyInput := &ec2.CopyImageInput{
		Name:          &amiCopyName,
		SourceImageId: &amiId,
		SourceRegion:  &c.region,
	}

	copyOutput, err := c.ec2Client.CopyImage(context.Background(), copyInput, func(o *ec2.Options) {
		o.Region = c.region
	})

	if err != nil {
		return "", err
	}

	if copyOutput.ImageId == nil {
		return "", fmt.Errorf("Image id for copied AMI not present")
	}

	return *copyOutput.ImageId, nil

}

func (c *client) GetMyAccountId() (string, error) {
	identityOutput, err := c.stsClient.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{}, func(o *sts.Options) {
		o.Region = c.region
	})

	if err != nil {
		return "", err
	}
	if identityOutput.Account == nil {
		return "", fmt.Errorf("Account id not found")
	}
	return *identityOutput.Account, nil
}

func (c *client) IsImageAvailable(amiId string) (bool, error) {
	image, err := c.FindGlobalImageById(amiId)
	if err != nil {
		return false, err
	} else if image.State == types.ImageStateAvailable {
		return true, nil
	}

	log.Printf("ami %s is in state %s, waiting for state %s", amiId, image.State, types.ImageStateAvailable)
	return false, nil
}

func (c *client) IsImageExported(s3Bucket string, s3Prefix string, amiId string, imageFormat string) (bool, error) {

	fullPrefix := s3Prefix + c.ExpectedImageS3Key(amiId, imageFormat)

	log.Printf("Searching for export in bucket %s with prefix %s", s3Bucket, fullPrefix)

	params := &s3.ListObjectsInput{
		Bucket: &s3Bucket,
		Prefix: &fullPrefix,
	}
	listOutput, err := c.s3Client.ListObjects(context.Background(), params, func(o *s3.Options) {
		o.Region = c.region
	})

	if err != nil {
		return false, err
	} else if len(listOutput.Contents) == 0 {
		return false, nil
	} else if len(listOutput.Contents) > 1 {
		return false, fmt.Errorf("found multiple matching s3 exports for ami %s", amiId)
	}

	return true, nil
}

func (c *client) WaitForImageToBecomeAvailable(amiId string, timeout time.Duration) error {
	ticker := time.NewTicker(timeout).C
	pollTicker := time.NewTicker(time.Second * 15).C

	available, _ := c.IsImageAvailable(amiId)
	if available {
		return nil
	}

	// if not available, poll until available or timeout is hit
	for {
		select {
		case <-ticker:
			return fmt.Errorf("timed out waiting for ami %s to become available", amiId)
		case <-pollTicker:
			log.Printf("Polling ami %s to determine if it is available", amiId)

			available, err := c.IsImageAvailable(amiId)
			if err != nil {
				log.Printf("err encountered looking up ami %s: %v", amiId, err)
				continue
			} else if available {
				log.Printf("ami %s is available", amiId)
				return nil
			}
		}
	}
}

func main() {

	var region string
	var amiId string
	var s3Bucket string
	var s3Prefix string

	flag.StringVar(&region, "region", "", "The AWS region the AMI resides in. NOTE: if the AMI is shared from another account, a copy of the AMI will be created in the client's account in order to import to KubeVirt")
	flag.StringVar(&amiId, "ami-id", "", "The ID of the ami to import")
	flag.StringVar(&s3Bucket, "s3-bucket", "", "The s3 bucket to use to store and deliver the AMI into kubevirt")
	flag.StringVar(&s3Prefix, "s3-prefix", "kubevirt-image-exports/", "The s3 directory prefix to use to store the AMIs being exported into kubevirt")
	flag.Parse()

	if amiId == "" {
		log.Fatalf("--ami-id is required")
	} else if s3Bucket == "" {
		log.Fatalf("--s3-bucket is required")
	}

	cli, err := NewClient(region)
	if err != nil {
		log.Fatalf("err encountered creation aws client: %v", err)
	}

	// STEPS
	// 1. Find AMI and determine who owns it
	// 2. Copy AMI to client's account if owned by another account and shared with client
	// 3. Export AMI to s3 bucket
	// 4. Import AMI to KubeVirt using Datavolume

	// ----------------
	// Step 1: Find AMI
	// ----------------
	image, err := cli.FindGlobalImageById(amiId)
	if err != nil {
		log.Fatalf("err encountered looking up ami %s: %v", amiId, err)
	} else if image.OwnerId == nil {
		log.Fatalf("Image is missing owner id")
	}
	imageOwnerAccount := *image.OwnerId
	myAccount, err := cli.GetMyAccountId()
	if err != nil {
		log.Fatalf("Unable to detect account id: %v", err)
	}

	// ----------------
	// Step 2: Copy AMI into client's account if owned by another account
	// ----------------
	amiToExport := ""
	if imageOwnerAccount == myAccount {
		log.Printf("Image is owned by client's account: %s", myAccount)
		amiToExport = amiId
	} else {
		log.Printf("Image is owned by another account %s. Client account is %s", imageOwnerAccount, myAccount)
		imageCopyName := cli.CopyImageName(amiId)
		imageCopy, exists, err := cli.FindImageByName(imageCopyName, myAccount)
		if err != nil {
			log.Fatalf("Error encountered while searching for image by name: %v", err)
		}
		if exists {
			// see if we've already created a copy
			if imageCopy.ImageId == nil {
				log.Fatalf("Image id is nil on ami describe")
			}
			amiToExport = *imageCopy.ImageId
			log.Printf("Found local copy of image named [%s] in client's account", amiToExport)
		} else {
			// if no copy exists, create it
			amiToExport, err = cli.CopyImage(amiId, imageCopyName)
			if err != nil {
				log.Fatalf("Error copying ami %s: %v", amiId, err)
			}
			log.Printf("Made copy of ami id %s in client's account. New ami copy is called [%s]", amiId, amiToExport)
		}
	}

	err = cli.WaitForImageToBecomeAvailable(amiToExport, time.Minute*15)
	if err != nil {
		log.Fatalf("Error encountered while waiting for ami %s to become available: %v", amiToExport, err)
	}

	// ----------------
	// Step 3: Export AMI to s3 bucket
	// ----------------

	exists, err := cli.IsImageExported(s3Bucket, s3Prefix, amiToExport, ExportImageFormat)
	if err != nil {
		log.Fatalf("Error encountered determining if image is exported to s3 bucket: %v", err)
	}

	// TODO
	// - Determine if task for ami already exists or not
	// - If task already exists, is it completed?
	// - If it is completed, does s3 export file still exist?
	//
	// If all of that is true, then skip the export step
	//
	//

	if !exists {
		log.Printf("Exporting ami %s to s3 bucket %s", amiToExport, s3Bucket)
		/*
			taskId, err := cli.ExportImage(amiToExport, s3Bucket, s3Prefix, exportImageFormat)
			if err != nil {
				log.Fatalf("Creation of export task for AMI %s to s3 failed: %v", amiToExport, err)
			}

			err = cli.WaitForExportImageCompletion(taskId, time.Minute*15)
			if err != nil {
				log.Fatalf("Exporting of AMI %s to s3 failed: %v", amiToExport, err)
			}
		*/
	} else {
		log.Printf("Found existing s3 export for ami %s", amiToExport)
	}
}
