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
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"net/http"
)

type CDController struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
	dataChannel chan *unstructured.Unstructured
	services.IService
}

func (s *CDController) Handle(addr string) {
	panic("implement me")
}

func (s *CDController) handle(cd *v1.CD) error {
	if cd.Spec.Done {
		return nil
	}
	return nil
}

func (s *CDController) recv(stop <-chan struct{}) error {
	gvr, err := s.GetGvr(k8s.CD)
	if err != nil {
		return err
	}
	_ = gvr

	list, err := s.List(common.YceCloudExtensions, k8s.CD, "", 0, 0, nil)
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		value := item
		cd := &v1.CD{}
		if err := tools.UnstructuredObjectToInstanceObj(&value, cd); err != nil {
			fmt.Printf("UnstructuredObjectToInstanceObj error (%s)", err)
			continue
		}
		if err := s.handle(cd); err != nil {
			fmt.Printf("handle ci error (%s)", err)
			continue
		}
	}
	cdList := &v1.CDList{}
	if err := tools.UnstructuredListObjectToInstanceObjectList(list, cdList); err != nil {
		return fmt.Errorf("UnstructuredListObjectToInstanceObjectList error (%s) (%v)", err, list)
	}

	eventChan, err := s.Watch(common.YceCloudExtensions, k8s.CD, cdList.GetResourceVersion(), 0, nil)
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
			cd := &v1.CD{}
			err := tools.RuntimeObjectToInstance(item.Object, cd)
			if err != nil {
				fmt.Printf("RuntimeObjectToInstance error (%s) (%v)", err, item.Object)
				continue
			}
			if err := s.handle(cd); err != nil {
				fmt.Printf("cd controller handle error (%s) (%v)", err, item.Object)
				continue
			}
		}
	}
}

func (s *CDController) Run(addr string, stop <-chan struct{}) error {
	route := gin.New()
	route.POST("/", func(g *gin.Context) {
		// 接收到 echoer post 的请求数据
		rawData, err := g.GetRawData()
		if err != nil {
			requestErr(g, err)
			return
		}
		request := &resource.RequestCd{}
		if err := json.Unmarshal(rawData, request); err != nil {
			requestErr(g, err)
			return
		}
		// 构造CD的参数
		if err := json.Unmarshal(rawData, request); err != nil {
			requestErr(g, err)
			return
		}

		// {git_project_name}-{Branch}
		project, err := tools.ExtractService(request.ServiceName)
		var name = fmt.Sprintf("%s-%s", project, request.DeployType)

		// 构造一个CI的结构
		cd := &v1.CD{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CD",
				APIVersion: "yamecloud.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: common.YceCloudExtensions,
			},
			Spec: v1.CDSpec{
				ServiceName:     &request.ServiceName,
				DeployNamespace: &request.DeployNamespace,
				ServiceImage:    &request.ServiceImage,
				ArtifactInfo:    request.ArtifactInfo,
				DeployType:      &request.DeployType,

				FlowId:     &request.FlowId,
				StepName:   &request.StepName,
				AckStates:  request.AckStates,
				UUID:       &request.UUID,
				RetryCount: &request.RetryCount,

				Done: false,
			},
		}
		// 转换成unstructured 类型
		unstructured, err := tools.InstanceToUnstructured(cd)
		if err != nil {
			requestErr(g, err)
			return
		}
		// 写入CRD配置
		obj, _, err := s.Apply(common.YceCloudExtensions, k8s.CD, name, unstructured)
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

func NewCDController(cfg *configure.InstallConfigure) Interface {
	return &CDController{
		InstallConfigure: cfg,
		IDataSource:      datasource.NewIDataSource(cfg),
	}
}
