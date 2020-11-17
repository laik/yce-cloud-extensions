package common

import (
	"flag"
	"fmt"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

var (
	// InCluster Flag for the application runtime
	InCluster bool
	// DefaultConfigFile is the default bootstrap configuration
	KubeConfig *string
	// YceCloudExtensions yce-cloud-extensions default namespace
	YceCloudExtensions = "yce-cloud-extensions"
	// YceCloudExtensionsOps yce-cloud-extensions is ci scheduler namespace
	YceCloudExtensionsOps = fmt.Sprintf("%s-%s", YceCloudExtensions, "ops")
	// Echoer server Address
	EchoerAddr = "http://127.0.0.1:8080/step"
)

const (
	WARN  = "[WARN]"
	INFO  = "[INFO]"
	ERROR = "[ERROR]"
)

func init() {
	flag.BoolVar(&InCluster, "incluster", false, "-incluster true")
	flag.StringVar(&EchoerAddr, "echoer", "http://127.0.0.1:8080/step", "-echoer http://127.0.0.1:8080/step")

	if home := homedir.HomeDir(); home != "" {
		KubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		KubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
}
