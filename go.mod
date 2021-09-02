module kubevirt.io/kubevirt-cloud-import

go 1.15

require (
	github.com/aws/aws-sdk-go-v2/config v1.6.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.14.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.13.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.6.2
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v12.0.0+incompatible
	kubevirt.io/client-go v0.44.1
	kubevirt.io/containerized-data-importer v1.38.0
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210817132244-67c28690af52
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/openshift/library-go => github.com/mhenriks/library-go v0.0.0-20200804184258-4fc3a5379c7a
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v1.0.2
)

// Pinned to v0.22.1
replace (
	k8s.io/api => k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.1
	k8s.io/apiserver => k8s.io/apiserver v0.22.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.1
	k8s.io/client-go => k8s.io/client-go v0.22.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.1
	k8s.io/code-generator => k8s.io/code-generator v0.22.1
	k8s.io/component-base => k8s.io/component-base v0.22.1
	k8s.io/cri-api => k8s.io/cri-api v0.22.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.1
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.1
	k8s.io/kubectl => k8s.io/kubectl v0.22.1
	k8s.io/kubelet => k8s.io/kubelet v0.22.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.1
	k8s.io/metrics => k8s.io/metrics v0.22.1
	k8s.io/node-api => k8s.io/node-api v0.22.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.1
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.22.1
	k8s.io/sample-controller => k8s.io/sample-controller v0.22.1
)

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.5 // indirect

exclude k8s.io/cluster-bootstrap v0.0.0
