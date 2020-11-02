package configure

import (
	"fmt"

	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

type RuntimeMode string

var AppRuntimeMode RuntimeMode = Default

func SetTheAppRuntimeMode(rm RuntimeMode) {
	AppRuntimeMode = rm
}

const (
	// InCluster when deploying in k8s, use this option
	InCluster RuntimeMode = "InCluster"
	// Default when deploying in non k8s, use this option and the is default option
	Default RuntimeMode = "Default"
)

// InstallConfigure ...
type InstallConfigure struct {
	// kubernetes reset config
	RestConfig *rest.Config
	// k8s CacheInformerFactory
	*k8s.CacheInformerFactory
	// k8s client
	*kubernetes.Clientset
}

func NewInstallConfigure(k8sResLister k8s.ResourceLister, k8sjsondata []byte) (*InstallConfigure, error) {
	var (
		clientSet      *kubernetes.Clientset
		resetConfig    *rest.Config
		clientv1Config *clientcmdapiv1.Config
		err            error
	)

	switch AppRuntimeMode {
	case Default:
		k8sConfig, err := k8s.CreateConfigFromJSON(k8sjsondata)
		if err != nil {
			return nil, err
		}
		clientv1Config = &k8sConfig.Config
		clientSet, resetConfig, err = k8s.BuildClientSet("", k8sConfig.Config)
	case InCluster:
		clientSet, resetConfig, err = k8s.CreateInClusterConfig()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("not define the runtime mode")
	}

	cacheInformerFactory, err := k8s.NewCacheInformerFactory(k8sResLister, resetConfig, clientv1Config)
	if err != nil {
		return nil, err
	}

	return &InstallConfigure{
		CacheInformerFactory: cacheInformerFactory,
		Clientset:            clientSet,
		RestConfig:           resetConfig,
	}, nil
}
