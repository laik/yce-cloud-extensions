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
	"github.com/laik/yce-cloud-extensions/pkg/services"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"net/http"
)

type CIController struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
	dataChannel chan *unstructured.Unstructured
	services.IService
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

func (s *CIController) handle(ci *v1.CI) error {
	if ci.Spec.Done {
		return nil
	}
	return nil
}

func (s *CIController) recv(stop <-chan struct{}) error {
	gvr, err := s.GetGvr(k8s.CI)
	if err != nil {
		return err
	}
	_ = gvr

	list, err := s.List(common.YceCloudExtensions, k8s.CI, "", 0, 0, nil)
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		value := item
		ci := &v1.CI{}
		if err := UnstructuredObjectToInstanceObj(&value, ci); err != nil {
			fmt.Printf("UnstructuredObjectToInstanceObj error (%s)", err)
			continue
		}
		if err := s.handle(ci); err != nil {
			fmt.Printf("handle ci error (%s)", err)
			continue
		}
	}
	ciList := &v1.CIList{}
	if err := UnstructuredListObjectToInstanceObjectList(list, ciList); err != nil {
		return fmt.Errorf("UnstructuredListObjectToInstanceObjectList error (%s) (%v)", err, list)
	}

	eventChan, err := s.Watch(common.YceCloudExtensions, k8s.CI, ciList.GetResourceVersion(), 0, nil)
	if err != nil {
		return fmt.Errorf("watch error (%s)", err)
	}

	for {
		select {
		case <-stop:
			return nil
		case item, ok := <-eventChan:
			if !ok {
				return nil
			}
			ci := &v1.CI{}
			err := RuntimeObjectToInstance(item.Object, ci)
			if err != nil {
				fmt.Printf("RuntimeObjectToInstance error (%s) (%v)", err, item.Object)
				continue
			}
			if err := s.handle(ci); err != nil {
				fmt.Printf("ci controller handle error (%s) (%v)", err, item.Object)
				continue
			}
		}
	}
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
				Done:       false,
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

	go s.Start(stop)
	go route.Run(addr)


	return s.recv(stop)
}

func NewCIController(cfg *configure.InstallConfigure) Interface {
	return &CIController{
		InstallConfigure: cfg,
		IDataSource:      datasource.NewIDataSource(cfg),
	}
}
