module github.com/laik/yce-cloud-extensions

go 1.15

require (
	github.com/ghodss/yaml v0.0.0-20150909031657-73d445a93680
	github.com/gin-gonic/gin v1.6.3
	github.com/go-resty/resty/v2 v2.3.0
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/onsi/ginkgo v1.12.3 // indirect
	github.com/tidwall/gjson v1.6.3
	github.com/tidwall/sjson v1.1.2
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/utils v0.0.0-20201027101359-01387209bb0d // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.1
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.3
	//google.golang.org/grpc => google.golang.org/grpc v1.26.0
	k8s.io/api => k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.0
	k8s.io/apiserver => k8s.io/apiserver v0.19.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.0
	k8s.io/client-go => k8s.io/client-go v0.19.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.0
	k8s.io/code-generator => k8s.io/code-generator v0.19.0
	k8s.io/component-base => k8s.io/component-base v0.19.0
	k8s.io/cri-api => k8s.io/cri-api v0.19.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.0
	k8s.io/kubectl => k8s.io/kubectl v0.19.0
	k8s.io/kubelet => k8s.io/kubelet v0.19.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.0
	k8s.io/metrics => k8s.io/metrics v0.19.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.0
)
