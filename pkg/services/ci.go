package services

import (
	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"github.com/laik/yce-cloud-extensions/pkg/datasource"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"k8s.io/apimachinery/pkg/api/errors"
)

var _ IService = &CIService{}
var taskTemplate string = ""

type CIService struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
}

func (c *CIService) Start(stop <-chan struct{}) {
	panic("implement me")
}

func (c *CIService) checkTaskResource(name string) error {
	obj, err := c.Get(common.YceCloudExtensions, k8s.Task, name)
	if !errors.IsNotFound(err) {
		return err
	}
	_ = obj
	if !compareSpec(taskTemplate, "") {
		//
	}
	return nil
}
