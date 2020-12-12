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
	servicescd "github.com/laik/yce-cloud-extensions/pkg/services/cd"
	httpclient "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"time"
)

type CDController struct {
	*configure.InstallConfigure
	datasource.IDataSource
	httpclient.IClient
	services.IService
	lastVersion string
}

func (s *CDController) handle(cd *v1.CD) error {
	resp := &resource.Response{
		FlowId:   *cd.Spec.FlowId,
		StepName: *cd.Spec.StepName,
		AckState: cd.Spec.AckStates[0],
		UUID:     *cd.Spec.UUID,
		Done:     cd.Spec.Done,
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

func (s *CDController) response2echoer(data map[string]interface{}) error {
	request := s.Post(common.EchoerAddr)
	for k, v := range data {
		request.Params(k, v)
	}
	if err := request.Do(); err != nil {
		return err
	}
	return nil
}

func (s *CDController) recv(stop <-chan struct{}) error {
	list, err := s.List(common.YceCloudExtensions, k8s.CD, "", 0, 0, nil)
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		value := item
		cd := &v1.CD{}
		if err := tools.UnstructuredObjectToInstanceObj(&value, cd); err != nil {
			fmt.Printf("%s UnstructuredObjectToInstanceObj error (%s)\n", common.ERROR, err)
			continue
		}
		if err := s.handle(cd); err != nil {
			fmt.Printf("%s handle cd error (%s)\n", common.ERROR, err)
			continue
		}
	}

	cdList := &v1.CDList{}
	if err := tools.UnstructuredListObjectToInstanceObjectList(list, cdList); err != nil {
		return fmt.Errorf("UnstructuredListObjectToInstanceObjectList error (%s) (%v)", err, list)
	}

RETRY:
	eventChan, err := s.Watch(common.YceCloudExtensions, k8s.CD, cdList.GetResourceVersion(), 0, nil)
	if err != nil {
		fmt.Printf("%s watch error (%s)\n", common.ERROR, err)
		time.Sleep(1 * time.Second)
		goto RETRY
	}

	fmt.Printf("cd controller start watch cd event\n")
	for {
		select {
		case <-stop:
			return nil
		case item, ok := <-eventChan:
			if !ok {
				goto RETRY
			}
			cd := &v1.CD{}
			err := tools.RuntimeObjectToInstance(item.Object, cd)
			if err != nil {
				fmt.Printf("%s RuntimeObjectToInstance error (%s)\n", common.ERROR, err)
				continue
			}
			if err := s.handle(cd); err != nil {
				fmt.Printf("%s cd controller handle error (%s)\n", common.ERROR, err)
				continue
			}
			s.lastVersion = cd.GetResourceVersion()
		}
	}
}

func (s *CDController) Run(addr string, stop <-chan struct{}) error {
	route := gin.New()
	route.Use(gin.Logger())

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

		artifactInfo := &v1.ArtifactInfo{
			Command:   make([]string, 0),
			Arguments: make([]string, 0),
		}
		if len(request.ArtifactInfo) > 0 {
			if err = json.Unmarshal([]byte(request.ArtifactInfo), artifactInfo); err != nil {
				requestErr(g, err)
				return
			}
		}

		var name = fmt.Sprintf("%s-%s", request.ServiceName, request.DeployType)
		name = strings.ToLower(strings.Replace(name, "_", "-", -1))

		// 构造一个CD的结构
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
				ArtifactInfo:    artifactInfo,
				DeployType:      &request.DeployType,
				CPULimit:        &request.CPULimit,
				MEMLimit:        &request.MEMLimit,
				CPURequests:     &request.CPURequests,
				MEMRequests:     &request.MEMRequests,
				Replicas:        request.Replicas,
				FlowId:          &request.FlowId,
				StepName:        &request.StepName,
				AckStates:       request.AckStates,
				UUID:            &request.UUID,
				Done:            false,
			},
		}
		// 转换成unstructured 类型
		unstructured, err := tools.InstanceToUnstructured(cd)
		if err != nil {
			requestErr(g, err)
			return
		}
		// 写入CRD配置
		obj, _, err := s.Apply(common.YceCloudExtensions, k8s.CD, name, unstructured, true)
		if err != nil {
			internalApplyErr(g, err)
			fmt.Printf("cd controller apply (%s) error (%s)\n", name, err)
			return
		}

		g.JSON(http.StatusOK, obj)
	})
	go s.Start(stop)

	go route.Run(addr)

	return s.recv(stop)
}

func NewCDController(cfg *configure.InstallConfigure) Interface {
	drs := datasource.NewIDataSource(cfg)
	return &CDController{
		InstallConfigure: cfg,
		IService:         servicescd.NewCDService(cfg, drs),
		IDataSource:      datasource.NewIDataSource(cfg),
		IClient:          httpclient.NewIClient(),
	}
}
