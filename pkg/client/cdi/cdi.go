package cdi

import (
	"context"
	"fmt"
	"os"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	cdiclient "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
)

type client struct {
	cdiClient *cdiclient.Clientset
}

func NewClient(master string, kubeconfig string) (*client, error) {

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	cfg, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}
	cdiClient, err := cdiclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &client{cdiClient: cdiClient}, nil
}

func (c *client) ImportFromS3IntoPvc(pvcName,
	pvcNamespace,
	pvcStorageClass,
	pvcAccessMode,
	s3Bucket,
	s3FilePath,
	s3SecretName string,
	storageQuantity resource.Quantity,

) error {
	dataVolume := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pvcNamespace,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: &cdiv1.DataVolumeSource{
				S3: &cdiv1.DataVolumeSourceS3{
					URL:       fmt.Sprintf("https://%s.s3.us-west-2.amazonaws.com/%s", s3Bucket, s3FilePath),
					SecretRef: s3SecretName,
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.PersistentVolumeAccessMode(pvcAccessMode)},
				Resources: k8sv1.ResourceRequirements{
					Requests: make(k8sv1.ResourceList),
				},
			},
		},
	}

	if pvcStorageClass != "" {
		dataVolume.Spec.PVC.StorageClassName = &pvcStorageClass
	}

	dataVolume.Spec.PVC.Resources.Requests[k8sv1.ResourceStorage] = storageQuantity
	_, err := c.cdiClient.CdiV1beta1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}
