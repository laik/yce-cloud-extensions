package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"github.com/laik/yce-cloud-extensions/pkg/datasource"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"net/http"
)

type CIController struct {
	*configure.InstallConfigure
	ds datasource.IDataSource
	c  client.IClient
}

func (s *CIController) Response(addr string, data map[string]interface{}) error {
	gvr, err := s.GetGvr(k8s.CI)
	if err != nil {
		return err
	}
	_ = gvr
	list, err := s.ds.List(common.YceCloudExtensions, k8s.CI, "", 0, 0, nil)
	if err != nil {
		return err
	}

	_ = list

	//s.ds.Watch(common.YceCloudExtensions,)
	request := s.c.Post(addr)
	for k, v := range data {
		request.Params(k, v)
	}
	return request.Do()
}

func (s *CIController) Run(addr string, stop <-chan struct{}) error {
	route := gin.New()
	route.POST("/", func(g *gin.Context) {
		// record action
		rawData, err := g.GetRawData()
		if err != nil {
			requestErr(g, err)
			return
		}
		_ = rawData
		g.JSON(http.StatusOK, "")
	})
	return route.Run(addr)
}

func NewCIController(cfg *configure.InstallConfigure) Interface {
	return &CIController{InstallConfigure: cfg}
}
