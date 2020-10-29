package controller

import (
	"github.com/gin-gonic/gin"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"net/http"
)

type CIController struct {
	c client.IClient
}

func (s *CIController) Response(addr string, data map[string]interface{}) error {
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
		g.JSON(http.StatusOK, "")
	})
	return route.Run(addr)
}

func NewCIController() Interface {
	return &CIController{}
}
