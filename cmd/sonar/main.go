package main

import (
	"flag"
	"fmt"

	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	ctl "github.com/laik/yce-cloud-extensions/pkg/controller"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
)

func needInit() (*configure.InstallConfigure, error) {
	if common.InCluster {
		configure.SetTheAppRuntimeMode(configure.InCluster)
	}
	cfg, err := configure.NewInstallConfigure(k8s.NewResources(nil))
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

var addr = "0.0.0.0:8080"

func main() {
	flag.StringVar(&addr, "addr", "0.0.0.0:8080", "-addr 0.0.0.0:8080")
	flag.Parse()
	cfg, err := needInit()
	if err != nil {
		panic(err)
	}
	fmt.Println(addr)
	if err := ctl.NewSonarController(cfg).Run(addr); err != nil {
		panic(err)
	}
}
