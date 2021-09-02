# kubevirt-cloud-import
Automation tools for importing Cloud Virtual Machines (from EC2, GCP, Azure) into a KubeVirt cluster using Tekton Pipelines.

# Importing AMI into KubeVirt

Automation for importing an AMI into KubeVirt works by exporting the AMI as a vdmk file into an s3 bucket then importing the vdmk file from s3 into a PVC using a DataVolume.

# Prerequisites 

Before importing an AMI via the cli command or using the Tekton task, the following prerequisites must be met.
- Create the [AWS vmimport service role](https://docs.aws.amazon.com/vm-import/latest/userguide/vmie_prereqs.html#vmimport-role) which is required in order to allow AWS to export an AMI to an s3 bucket on your behalf.
- Create an S3 bucket that will be used to export the AMI to KubeVirt
- Create an access credential secret in the k8s that gives permission to retrieve data from the s3 bucket your AMI will be stored in.

Below is an example of how the access credential secret is formatted.
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: s3-readonly-cred
  labels:
    app: containerized-data-importer
type: Opaque
data:
  accessKeyId: ""  # <optional: your key or user name, base64 encoded>
  secretKey:    "" # <optional: your secret or password, base64 encoded>
```
## CLI AMI Import

```
export S3_BUCKET=my-bucket
export S3_SECRET=s3-readonly-cred
export AWS_REGION=us-west-2
export AMI_ID=ami-00a4fdd3db8bb2851
export PVC_STORAGECLASS=rook-ceph-block
export PVC_NAME=fedora34-golden-image

ami-export-s3 --s3-bucket $S3_BUCKET --region $AWS_REGION --ami-id $AMI_ID --pvc-storageclass $PVC_STORAGECLASS --s3-secret $S3_SECRET --pvc-name $PVC_NAME
```

Example Output

```
$ ./ami-export-s3 --s3-bucket $S3_BUCKET --region $AWS_REGION --ami-id $AMI_ID --pvc-storageclass $PVC_STORAGECLASS --s3-secret $S3_SECRET --pvc-name $PVC_NAME
2021/09/02 17:02:44 Image is owned by another account 125523088429. Client account is 269733383066
2021/09/02 17:02:45 Found local copy of image named [ami-0d8e0766632b22bc0] in client's account
2021/09/02 17:02:45 Found existing s3 export for ami ami-0d8e0766632b22bc0
2021/09/02 17:02:45 AMI is exported to s3 bucket: [my-bucket] at file path [kubevirt-image-exports/orig-ami-0d8e0766632b22bc0-export-ami-0a98ec99f7e1bcc65.vmdk]
2021/09/02 17:02:45 Created DataVolume to import AMI [ami-00a4fdd3db8bb2851] to pvc [default/fedora34-golde-image]
2021/09/02 17:02:45 Polling DataVolume default/fedora34-golden-image to determine if import is completed
2021/09/02 17:02:45 Success! AMI [ami-00a4fdd3db8bb2851] imported into PVC [default/fedora34-golden-image]
```

## Tekton AMI Import

WIP

