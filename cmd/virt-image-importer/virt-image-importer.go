package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var (
	region string
	amiId  string
)

func init() {
	flag.StringVar(&region, "region", "", "The AWS region.")
	flag.StringVar(&amiId, "ami-id", "", "The ID of the ami to import")
}

func main() {
	flag.Parse()

	if amiId == "" {
		log.Fatalf("--ami-id is required")
	}

	// Load the SDK's configuration from environment and shared config, and
	// create the ec2Client with this.
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load SDK configuration, %v", err)
	}

	if region == "" {
		region = cfg.Region
	}

	ec2Client := ec2.NewFromConfig(cfg)
	stsClient := sts.NewFromConfig(cfg)

	params := &ec2.DescribeImagesInput{
		ImageIds: []string{amiId},
	}

	amiListOutput, err := ec2Client.DescribeImages(context.Background(), params, func(o *ec2.Options) {
		o.Region = region
	})
	if err != nil {
		log.Fatalf("Failed to retrieve ami list: %v", err)
	}

	identityOutput, err := stsClient.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{}, func(o *sts.Options) {
		o.Region = region
	})

	fmt.Printf("images found: %d\n", len(amiListOutput.Images))
	fmt.Printf("metadata: %v\n", amiListOutput.ResultMetadata)
	for _, image := range amiListOutput.Images {
		if image.ImageId != nil {
			log.Printf("image: %s\n", *image.ImageId)
		}

		if image.OwnerId != nil {
			log.Printf("image owner: %s\n", *image.OwnerId)
		}
	}

	if identityOutput.Account != nil {
		fmt.Printf("My account id: %s\n", *identityOutput.Account)
	}

	// TODO copy if
	// * ami account doesn't equal current account
	// * regions don't match
	// Use name as unique id, name of AMI copy is immutable which means we can use it to determine
	// if the copy already took place
}
