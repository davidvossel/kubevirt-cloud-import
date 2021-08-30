package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"kubevirt.io/kubevirt-cloud-import/pkg/client/aws"
)

const (
	ExportImageFormat = "vmdk"
	S3PrefixFormat    = "kubevirt-image-exports/orig-%s-"
)

func main() {

	var region string
	var amiId string
	var s3Bucket string

	flag.StringVar(&region, "region", "", "The AWS region the AMI resides in. NOTE: if the AMI is shared from another account, a copy of the AMI will be created in the client's account in order to import to KubeVirt")
	flag.StringVar(&amiId, "ami-id", "", "The ID of the ami to import")
	flag.StringVar(&s3Bucket, "s3-bucket", "", "The s3 bucket to use to store and deliver the AMI into kubevirt")
	flag.Parse()

	if amiId == "" {
		log.Fatalf("--ami-id is required")
	} else if s3Bucket == "" {
		log.Fatalf("--s3-bucket is required")
	}

	cli, err := aws.NewClient(region)
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
	foundS3Bucket, foundS3FilePath, completed, exists, err := cli.GetExportTaskStatus("", amiToExport, ExportImageFormat)
	if !exists {
		log.Printf("Exporting ami %s to s3 bucket %s", amiToExport, s3Bucket)
		s3Prefix := fmt.Sprintf(S3PrefixFormat, amiToExport)

		taskId, err := cli.ExportImage(amiToExport, s3Bucket, s3Prefix, ExportImageFormat)
		if err != nil {
			log.Fatalf("Creation of export task for AMI %s to s3 failed: %v", amiToExport, err)
		}

		foundS3Bucket, foundS3FilePath, err = cli.WaitForExportImageCompletion(amiToExport, taskId, ExportImageFormat, time.Minute*15)
		if err != nil {
			log.Fatalf("Exporting of AMI %s to s3 failed: %v", amiToExport, err)
		}
	} else if !completed {
		log.Printf("Waiting for existing image export job to complete")
		foundS3Bucket, foundS3FilePath, err = cli.WaitForExportImageCompletion(amiToExport, "", ExportImageFormat, time.Minute*15)
		if err != nil {
			log.Fatalf("Exporting of AMI %s to s3 failed: %v", amiToExport, err)
		}
	} else {
		log.Printf("Found existing s3 export for ami %s", amiToExport)
	}

	log.Printf("AMI is exported to s3 bucket: [%s] at file path [%s]", foundS3Bucket, foundS3FilePath)

}
