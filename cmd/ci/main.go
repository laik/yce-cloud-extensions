package main

import (
	"flag"

	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	ctl "github.com/laik/yce-cloud-extensions/pkg/controller"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
)

func needInit(data []byte) error {
	if common.INCLUSTER {
		configure.SetTheAppRuntimeMode(configure.InCluster)
	}
	_, err := configure.NewInstallConfigure(k8s.NewResources(nil), data)
	if err != nil {
		return err
	}
	return nil
}

func main() {

	flag.Parse()

	srv := ctl.NewCIController()
	stop := make(chan struct{})
	if err := srv.Run(":8080", stop); err != nil {
		panic(err)
	}
}
