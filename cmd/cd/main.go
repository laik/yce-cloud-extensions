package main

import (
	"flag"

	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	ctl "github.com/laik/yce-cloud-extensions/pkg/controller"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
)

func needInit() (*configure.InstallConfigure, error) {
	if common.InCluster {
		configure.SetTheAppRuntimeMode(configure.InCluster)
	}
	cfg, err := configure.NewInstallConfigure(k8s.NewResources([]string{
		k8s.CI,
		k8s.Pipeline,
		k8s.PipelineRun,
		k8s.Task,
		k8s.TaskRun,
		k8s.PipelineResource,
		k8s.TektonGraph,
		k8s.TektonConfig,
		k8s.SONAR,
		k8s.UNIT,
	}))
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

	if err := ctl.NewCDController(cfg).Run(addr); err != nil {
		panic(err)
	}
}
