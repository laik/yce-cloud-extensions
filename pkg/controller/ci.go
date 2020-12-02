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
	servicesci "github.com/laik/yce-cloud-extensions/pkg/services/ci"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	httpclient "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"time"
)

type CIController struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
	services.IService
	lastVersion string
}

func (s *CIController) response2echoer(data map[string]interface{}) error {
	request := s.Post(common.EchoerAddr)
	for k, v := range data {
		request.Params(k, v)
	}
	if err := request.Do(); err != nil {
		return err
	}
	return nil
}

func (s *CIController) reconcile(ci *v1.CI) error {
	if !ci.Spec.Done {
		return nil
	}
	if len(ci.Spec.AckStates) == 0 {
		return nil
	}
	resp := &resource.Response{
		FlowId:   *ci.Spec.FlowId,
		StepName: *ci.Spec.StepName,
		AckState: ci.Spec.AckStates[0],
		UUID:     *ci.Spec.UUID,
		Done:     ci.Spec.Done,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(respBytes, &data)
	if err != nil {
		return err
	}

	return s.response2echoer(data)
}

func (s *CIController) recv(stop <-chan struct{}) error {
	list, err := s.List(common.YceCloudExtensionsOps, k8s.CI, "", 0, 0, nil)
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		value := item
		ci := &v1.CI{}
		if err := tools.UnstructuredObjectToInstanceObj(&value, ci); err != nil {
			fmt.Printf("UnstructuredObjectToInstanceObj error (%s)", err)
			continue
		}
		if err := s.reconcile(ci); err != nil {
			fmt.Printf("%s handle ci error (%s)\n", common.ERROR, err)
			continue
		}
	}
	ciList := &v1.CIList{}
	if err := tools.UnstructuredListObjectToInstanceObjectList(list, ciList); err != nil {
		return fmt.Errorf("UnstructuredListObjectToInstanceObjectList error (%s) (%v)", err, list)
	}

RETRY:
	eventChan, err := s.Watch(common.YceCloudExtensionsOps, k8s.CI, ciList.GetResourceVersion(), 0, nil)
	if err != nil {
		fmt.Printf("watch error (%s)\n", err)
		time.Sleep(1 * time.Second)
		goto RETRY
	}

	for {
		select {
		case <-stop:
			return nil
		case item, ok := <-eventChan:
			if !ok {
				goto RETRY
			}
			ci := &v1.CI{}
			err := tools.RuntimeObjectToInstance(item.Object, ci)
			if err != nil {
				fmt.Printf("%s RuntimeObjectToInstance error (%s)\n", common.WARN, err)
				continue
			}
			if err := s.reconcile(ci); err != nil {
				fmt.Printf("%s ci controller handle error (%s)\n", common.ERROR, err)
				continue
			}
			s.lastVersion = ci.GetResourceVersion()
		}
	}
}

func (s *CIController) Run(addr string, stop <-chan struct{}) error {
	gin.SetMode("debug")
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

		// {git-project-name}-{Branch}
		project, err := tools.ExtractProject(request.GitUrl)
		var name = fmt.Sprintf("%s-%s", project, request.Branch)

		// 构造一个CI的结构
		ci := &v1.CI{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CI",
				APIVersion: "yamecloud.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: common.YceCloudExtensionsOps,
			},
			Spec: v1.CISpec{
				GitURL:     &request.GitUrl,
				Branch:     &request.Branch,
				CommitID:   &request.CommitID,
				RetryCount: &request.RetryCount,
				Output:     &request.Output,
				CodeType:   &request.CodeType,
				FlowId:     &request.FlowId,
				StepName:   &request.StepName,
				AckStates:  request.AckStates,
				UUID:       &request.UUID,
				Done:       false,
			},
		}
		// 转换成unstructured 类型
		unstructured, err := tools.InstanceToUnstructured(ci)
		if err != nil {
			requestErr(g, err)
			return
		}
		// 写入CRD配置
		obj, _, err := s.Apply(common.YceCloudExtensionsOps, k8s.CI, name, unstructured, true)
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
	drs := datasource.NewIDataSource(cfg)
	return &CIController{
		InstallConfigure: cfg,
		IService:         servicesci.NewService(cfg, drs),
		IClient:          httpclient.NewIClient(),
		IDataSource:      drs,
	}
}
