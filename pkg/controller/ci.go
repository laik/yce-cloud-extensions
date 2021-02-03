package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"github.com/laik/yce-cloud-extensions/pkg/datasource"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	"github.com/laik/yce-cloud-extensions/pkg/proc"
	"github.com/laik/yce-cloud-extensions/pkg/resource"
	"github.com/laik/yce-cloud-extensions/pkg/services"
	servicesci "github.com/laik/yce-cloud-extensions/pkg/services/ci"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	httpclient "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CIController struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
	services.IService
	lastVersion string

	proc *proc.Proc
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
	if !ci.Spec.Done || len(ci.Spec.AckStates) == 0 {
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

func (s *CIController) recv(stop <-chan struct{}, errC chan<- error) {
	list, err := s.List(common.YceCloudExtensionsOps, k8s.CI, "", 0, 0, nil)
	if err != nil {
		errC <- err
		return
	}

	for _, item := range list.Items {
		value := item
		ci := &v1.CI{}
		if err := tools.UnstructuredObjectToInstanceObj(&value, ci); err != nil {
			fmt.Printf("%s UnstructuredObjectToInstanceObj error (%s)", common.ERROR, err)
			continue
		}
		if err := s.reconcile(ci); err != nil {
			fmt.Printf("%s handle ci error (%s)\n", common.ERROR, err)
			continue
		}
	}

	ciList := &v1.CIList{}
	if err := tools.UnstructuredListObjectToInstanceObjectList(list, ciList); err != nil {
		errC <- fmt.Errorf("UnstructuredListObjectToInstanceObjectList error (%s) (%v)", err, list)
		return
	}

	eventChan, err := s.Watch(common.YceCloudExtensionsOps, k8s.CI, ciList.GetResourceVersion(), 0, nil)
	if err != nil {
		errC <- err
		return
	}

	fmt.Printf("%s ci controller start watch ci channel.....\n", common.INFO)

	for {
		select {
		case <-stop:
			fmt.Printf("%s ci controller stop\n", common.INFO)
			return

		case item, ok := <-eventChan:
			if !ok {
				fmt.Printf("%s ci controller watch stone resource channel stop\n", common.ERROR)
				errC <- fmt.Errorf("controller watch ci channel closed")
				return
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

func (s *CIController) checkAndReconcileCi(name string) error {
	obj, err := s.Get(common.YceCloudExtensionsOps, k8s.CI, name)
	if err != nil {
		return err
	}
	ci := &v1.CI{}
	if err := tools.UnstructuredObjectToInstanceObj(obj, ci); err != nil {
		return err
	}
	if ci.Spec.Done == false {
		ci.Spec.AckStates = append(ci.Spec.AckStates, v1.FailState)
		ci.Spec.Done = true
		ciUnstructured, err := tools.InstanceToUnstructured(ci)
		if err != nil {
			return err
		}
		if _, _, err := s.Apply(common.YceCloudExtensionsOps, k8s.CI, name, ciUnstructured, false); err != nil {
			return err
		}
	}
	return nil
}

func (s *CIController) Run(addr string) error {
	gin.SetMode("debug")
	route := gin.New()
	route.Use(gin.Logger())

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
		var name = strings.ToLower(strings.Replace(fmt.Sprintf("%s-%s", project, request.Branch), "_", "-", -1))
		if len(request.ServiceName) > 0 {
			name = strings.ToLower(strings.Replace(fmt.Sprintf("%s-%s", request.ServiceName, name), "_", "-", -1))
		}
		name = strings.ToLower(strings.Replace(name, ".", "-", -1))

		if len(name) > 62 {
			name = name[len(name)-62:]
		}
		err = s.checkAndReconcileCi(name)
		if err != nil {
			fmt.Printf("check last ci error%s", err)
		}

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
				GitURL:      &request.GitUrl,
				Branch:      &request.Branch,
				CommitID:    &request.CommitID,
				RetryCount:  &request.RetryCount,
				Output:      &request.Output,
				CodeType:    request.CodeType,
				FlowId:      &request.FlowId,
				StepName:    &request.StepName,
				AckStates:   request.AckStates,
				UUID:        &request.UUID,
				ProjectPath: request.ProjectPath,
				ProjectFile: request.ProjectFile,
				Done:        false,
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
			fmt.Printf("ci controller apply (%s) error (%s)\n", name, err)
			return
		}

		g.JSON(http.StatusOK, obj)
	})

	go func() {
		s.proc.Error() <- route.Run(addr)
	}()

	s.proc.Add(s.Start)
	s.proc.Add(s.recv)

	return <-s.proc.Start()
}

func NewCIController(cfg *configure.InstallConfigure) Interface {
	drs := datasource.NewIDataSource(cfg)
	return &CIController{
		InstallConfigure: cfg,
		IService:         servicesci.NewService(cfg, drs),
		IClient:          httpclient.NewIClient(),
		IDataSource:      drs,

		proc: proc.NewProc(),
	}
}
