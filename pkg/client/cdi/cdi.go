package cdi

import "k8s.io/client-go/tools/clientcmd"

func NewCdiClient(master string, kubeconfig string) (*cdiclientset.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}
	cdiClient, err := cdiclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return cdiClient, nil
}
