package cdi

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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
	s3Region,
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
					URL:       fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s3Bucket, s3Region, s3FilePath),
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

func (c *client) getDataVolumePhase(name string, namespace string) (cdiv1.DataVolumePhase, error) {

	dv, err := c.cdiClient.CdiV1beta1().DataVolumes(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return cdiv1.PhaseUnset, err
	}

	return dv.Status.Phase, nil

}

func (c *client) WaitForS3ImportCompletion(pvcName string, pvcNamespace string, timeout time.Duration) error {
	var completed bool
	ticker := time.NewTicker(timeout).C
	pollTicker := time.NewTicker(time.Second * 15).C

	fn := func() (bool, error) {
		log.Printf("Polling DataVolume %s/%s to determine if import is completed", pvcNamespace, pvcName)
		phase, err := c.getDataVolumePhase(pvcName, pvcNamespace)
		if err != nil {
			return false, err
		} else if phase == cdiv1.Succeeded {
			return true, nil
		} else if phase == cdiv1.Failed {
			return false, fmt.Errorf("DataVolume %s/%s failed to import into pvc", pvcNamespace, pvcName)
		}
		return false, nil
	}

	completed, err := fn()
	if err != nil {
		return err
	} else if completed {
		return nil
	}

	// if not available, poll until available or timeout is hit
	for {
		select {
		case <-ticker:
			return fmt.Errorf("timed out waiting for datavolume %s/%s to complete", pvcNamespace, pvcName)
		case <-pollTicker:
			completed, err := fn()
			if err != nil {
				return err
			} else if completed {
				return nil
			}
		}
	}
}
