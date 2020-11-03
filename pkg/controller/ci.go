package controller

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"github.com/laik/yce-cloud-extensions/pkg/datasource"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	"github.com/laik/yce-cloud-extensions/pkg/resource"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"net/http"
	"strings"
)

type CIController struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
	dataChannel chan *unstructured.Unstructured
}

func (s *CIController) Handle(addr string) {
	var data map[string]interface{}
	request := s.Post(addr)

	for k, v := range data {
		request.Params(k, v)
	}
	if err := request.Do(); err != nil {
		panic(err)
	}
}

func (s *CIController) recv() error {
	gvr, err := s.GetGvr(k8s.CI)
	if err != nil {
		return err
	}
	_ = gvr

	list, err := s.List(common.YceCloudExtensions, k8s.CI, "", 0, 0, nil)
	if err != nil {
		return err
	}

	_ = list

	return nil
}

func (s *CIController) Run(addr string, stop <-chan struct{}) error {
	route := gin.New()
	route.POST("/", func(g *gin.Context) {
		// 接收到 echoer post 的请求数据
		rawData, err := g.GetRawData()
		if err != nil {
			requestErr(g, err)
			return
		}
		request := &resource.Request{}
		if err := json.Unmarshal(rawData, request); err != nil {
			requestErr(g, err)
			return
		}
		// 构造CI的参数
		if err := json.Unmarshal(rawData, request); err != nil {
			requestErr(g, err)
			return
		}

		// {git_project_name}-{Branch}
		project, err := extractProject(request.GitUrl)
		var name = fmt.Sprintf("%s-%s", project, request.Branch)

		// 构造一个CI的结构
		ci := &v1.CI{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CI",
				APIVersion: "yamecloud.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: common.YceCloudExtensions,
			},
			Spec: v1.CISpec{
				GitURL:     &request.GitUrl,
				Branch:     &request.Branch,
				CommitID:   &request.CommitID,
				RetryCount: &request.RetryCount,
				Output:     &request.Output,
				FlowId:     &request.FlowId,
				StepName:   &request.StepName,
				AckStates:  request.AckStates,
				UUID:       &request.UUID,
			},
		}
		// 转换成unstructured 类型
		unstructured, err := InstanceToUnstructured(ci)
		if err != nil {
			requestErr(g, err)
			return
		}
		// 写入CRD配置
		obj, _, err := s.Apply(common.YceCloudExtensions, k8s.CI, name, unstructured)
		if err != nil {
			internalApplyErr(g, err)
			return
		}

		g.JSON(http.StatusOK, obj)
	})

	go s.recv()

	return route.Run(addr)
}

func NewCIController(cfg *configure.InstallConfigure) Interface {
	return &CIController{
		InstallConfigure: cfg,
		IDataSource:      datasource.NewIDataSource(cfg),
	}
}

func extractProject(git string) (string, error) {
	if !strings.HasSuffix(git, ".git") {
		return "", fmt.Errorf("git addr illegal (%s)", git)
	}

	_slice := strings.Split(strings.TrimSuffix(git, ".git"), "/")
	if len(_slice) < 1 {
		return "", fmt.Errorf("git addr illegal (%s)", git)
	}
	return _slice[len(_slice)-1], nil
}
