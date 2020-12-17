package configure

import (
	"fmt"

	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	client "k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
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
	client.Interface
	// ResourceLister resource lister
	k8s.ResourceLister
}

func NewInstallConfigure(k8sResLister k8s.ResourceLister) (*InstallConfigure, error) {
	var (
		cli         client.Interface
		resetConfig *rest.Config
		err         error
	)

	switch AppRuntimeMode {
	case Default:
		cli, resetConfig, err = k8s.BuildClientSet(*common.KubeConfig)
	case InCluster:
		_, resetConfig, err = k8s.CreateInClusterConfig()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("not define the runtime mode")
	}

	cacheInformerFactory, err := k8s.NewCacheInformerFactory(k8sResLister, resetConfig)
	if err != nil {
		return nil, err
	}

	return &InstallConfigure{
		CacheInformerFactory: cacheInformerFactory,
		Interface:            cli,
		RestConfig:           resetConfig,
		ResourceLister:       k8sResLister,
	}, nil
}
