This is a WORK IN PROGRESS

# kubevirt-cloud-import
Automation tools for importing Cloud Virtual Machines (from EC2, GCP, Azure) into a KubeVirt cluster using Tekton Pipelines.

# Importing AMI into KubeVirt

Automation for importing an AMI into KubeVirt works by exporting the AMI as a vdmk file into an s3 bucket then importing the vdmk file from s3 into a PVC using a DataVolume.

# Prerequisites 

Before importing an AMI via the cli command or using the Tekton task, the following prerequisites must be met.
- Create the [AWS vmimport service role](https://docs.aws.amazon.com/vm-import/latest/userguide/vmie_prereqs.html#vmimport-role) which is required in order to allow AWS to export an AMI to an s3 bucket on your behalf.
- Create an S3 bucket that will be used to export the AMI to KubeVirt
- Create an access credential secret in the k8s that gives permission to...
	- retrieve data from the s3 bucket your AMI will be stored in
	- execute the export-image command

More info related to the AWS export-image functionality can be found [here](https://docs.aws.amazon.com/vm-import/latest/userguide/vmexport_image.html)

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

import-ami --s3-bucket $S3_BUCKET --region $AWS_REGION --ami-id $AMI_ID --pvc-storageclass $PVC_STORAGECLASS --s3-secret $S3_SECRET --pvc-name $PVC_NAME
```

Example Output

```
$ ./import-ami --s3-bucket $S3_BUCKET --region $AWS_REGION --ami-id $AMI_ID --pvc-storageclass $PVC_STORAGECLASS --s3-secret $S3_SECRET --pvc-name $PVC_NAME
2021/09/02 17:02:44 Image is owned by another account 125523088429. Client account is 269733383066
2021/09/02 17:02:45 Found local copy of image named [ami-0d8e0766632b22bc0] in client's account
2021/09/02 17:02:45 Found existing s3 export for ami ami-0d8e0766632b22bc0
2021/09/02 17:02:45 AMI is exported to s3 bucket: [my-bucket] at file path [kubevirt-image-exports/orig-ami-0d8e0766632b22bc0-export-ami-0a98ec99f7e1bcc65.vmdk]
2021/09/02 17:02:45 Created DataVolume to import AMI [ami-00a4fdd3db8bb2851] to pvc [default/fedora34-golde-image]
2021/09/02 17:02:45 Polling DataVolume default/fedora34-golden-image to determine if import is completed
2021/09/02 17:02:45 Success! AMI [ami-00a4fdd3db8bb2851] imported into PVC [default/fedora34-golden-image]
```

## Tekton AMI Import

**Step 1: Install Tekton + Tekton Tasks**

```bash
# install tekton
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml

# install kubevirt tekton tasks
export kv_tekton_release=v0.3.1
k apply -f https://github.com/kubevirt/kubevirt-tekton-tasks/releases/download/${kv_tekton_release}/kubevirt-tekton-tasks-kubernetes.yaml 

# install cloud import tekton tasks
kubectl apply --filename https://raw.githubusercontent.com/davidvossel/kubevirt-cloud-import/main/tasks/import-ami/manifests/import-ami.yaml 

```

**Step 2: Create a Pipeline**

An example pipeline can be found [here](https://raw.githubusercontent.com/davidvossel/kubevirt-cloud-import/main/examples/create-vm-from-ami-pipeline.yaml)
```bash
kubectl apply --filename https://raw.githubusercontent.com/davidvossel/kubevirt-cloud-import/main/examples/create-vm-from-ami-pipeline.yaml
```

Create a pipeline run. An example can be found [here](https://raw.githubusercontent.com/davidvossel/kubevirt-cloud-import/main/examples/create-vm-from-ami-pipeline-run.yaml)

**Step 3: Execute the pipeline run**


Post the pipeline run, which will kick off all the automation of importing the AMI and creating the VM
```bash
kubectl -apply --filename https://raw.githubusercontent.com/davidvossel/kubevirt-cloud-import/main/examples/create-vm-from-ami-pipeline-run.yaml
```

Watch for the pipeline run to complete

```bash
$ kubectl get pipelinerun -n kubevirt
selecting docker as container runtime
NAME                      SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
my-vm-creation-pipeline   True        Succeeded   11m         9m54s
```

Observe that the VM's VMI is online and running
```bash
$ kubectl get vmi -n kubevirt
selecting docker as container runtime
NAME          AGE   PHASE     IP               NODENAME   READY
vm-fedora34   11m   Running   10.244.196.175   node01     True

```

