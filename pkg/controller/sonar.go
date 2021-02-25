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
	"github.com/laik/yce-cloud-extensions/pkg/proc"
	"github.com/laik/yce-cloud-extensions/pkg/resource"
	"github.com/laik/yce-cloud-extensions/pkg/services"
	servicessonar "github.com/laik/yce-cloud-extensions/pkg/services/sonar"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	httpclient "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

type SonarController struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
	services.IService
	lastVersion string

	proc *proc.Proc
}

func (s *SonarController) response2echoer(data map[string]interface{}) error {
	request := s.Post(common.EchoerAddr)
	for k, v := range data {
		request.Params(k, v)
	}
	if err := request.Do(); err != nil {
		return err
	}
	return nil
}


func (s *SonarController) reconcile(sonar *v1.Sonar) error {
	if !sonar.Spec.Done || len(sonar.Spec.AckStates) == 0 {
		return nil
	}

	resp := &resource.Response{
		FlowId:   *sonar.Spec.FlowId,
		StepName: *sonar.Spec.StepName,
		AckState: sonar.Spec.AckStates[0],
		UUID:     *sonar.Spec.UUID,
		Done:     sonar.Spec.Done,
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

func (s *SonarController) recv(stop <-chan struct{}, errC chan<- error) {
	list, err := s.List(common.YceCloudExtensionsOps, k8s.SONAR, "", 0, 0, nil)
	if err != nil {
		errC <- err
		return
	}

	for _, item := range list.Items {
		value := item
		unit := &v1.Sonar{}
		if err := tools.UnstructuredObjectToInstanceObj(&value, unit); err != nil {
			fmt.Printf("%s UnstructuredObjectToInstanceObj error (%s)", common.ERROR, err)
			continue
		}
		if err := s.reconcile(unit); err != nil {
			fmt.Printf("%s handle sonar error (%s)\n", common.ERROR, err)
			continue
		}
	}

	sonarList := &v1.SonarList{}
	if err := tools.UnstructuredListObjectToInstanceObjectList(list, sonarList); err != nil {
		errC <- fmt.Errorf("UnstructuredListObjectToInstanceObjectList error (%s) (%v)", err, list)
		return
	}

	eventChan, err := s.Watch(common.YceCloudExtensionsOps, k8s.SONAR, sonarList.GetResourceVersion(), 0, nil)
	if err != nil {
		errC <- err
		return
	}

	fmt.Printf("%s sonar controller start watch unit channel.....\n", common.INFO)

	for {
		select {
		case <-stop:
			fmt.Printf("%s sonar controller stop\n", common.INFO)
			return

		case item, ok := <-eventChan:
			if !ok {
				fmt.Printf("%s sonar controller watch stone resource channel stop\n", common.ERROR)
				errC <- fmt.Errorf("controller watch sonar channel closed")
				return
			}

			sonar := &v1.Sonar{}
			err := tools.RuntimeObjectToInstance(item.Object, sonar)
			if err != nil {
				fmt.Printf("%s RuntimeObjectToInstance error (%s)\n", common.WARN, err)
				continue
			}

			if err := s.reconcile(sonar); err != nil {
				fmt.Printf("%s sonar controller handle error (%s)\n", common.ERROR, err)
				continue
			}

			s.lastVersion = sonar.GetResourceVersion()
		}
	}
}

func (s *SonarController) checkAndReconcileSonar(name string) error {
	obj, err := s.Get(common.YceCloudExtensionsOps, k8s.SONAR, name)
	if err != nil {
		return err
	}
	sonar := &v1.Sonar{}
	if err := tools.UnstructuredObjectToInstanceObj(obj, sonar); err != nil {
		return err
	}
	if sonar.Spec.Done == false {
		sonar.Spec.AckStates = append(sonar.Spec.AckStates, v1.FailState)
		sonar.Spec.Done = true
		sonarUnstructured, err := tools.InstanceToUnstructured(sonar)
		if err != nil {
			return err
		}
		if _, _, err := s.Apply(common.YceCloudExtensionsOps, k8s.SONAR, name, sonarUnstructured, false); err != nil {
			return err
		}
	}
	return nil
}

func (s *SonarController) Run(addr string) error {
	route := gin.New()
	route.Use(gin.Logger())

	route.POST("/", func(g *gin.Context) {
		// 接收到 echoer post 的请求数据
		rawData, err := g.GetRawData()
		if err != nil {
			requestErr(g, err)
			return
		}
		request := &resource.RequestSonar{}
		if err := json.Unmarshal(rawData, request); err != nil {
			requestErr(g, err)
			return
		}
		// 构造sonar的参数
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

		name = sonarPipelineRunName(name)

		err = s.checkAndReconcileSonar(name)
		if err != nil {
			fmt.Printf("check last sonar error %s", err)
		}

		// 构造一个sonar的结构
		sonar := &v1.Sonar{
			TypeMeta: metav1.TypeMeta{
				Kind:       "SONAR",
				APIVersion: "yamecloud.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: common.YceCloudExtensionsOps,
			},
			Spec: v1.SonarSpec{
				GitURL:   &request.GitUrl,
				Branch:   &request.Branch,
				Language: &request.Language,

				FlowId:    &request.FlowId,
				StepName:  &request.StepName,
				AckStates: request.AckStates,
				UUID:      &request.UUID,
				Done:      false,
			},
		}
		// 转换成unstructured 类型
		unstructured, err := tools.InstanceToUnstructured(sonar)
		if err != nil {
			requestErr(g, err)
			return
		}
		// 写入CRD配置
		obj, _, err := s.Apply(common.YceCloudExtensionsOps, k8s.SONAR, name, unstructured, true)
		if err != nil {
			internalApplyErr(g, err)
			fmt.Printf("sonar controller apply (%s) error (%s)\n", name, err)
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

func NewSonarController(cfg *configure.InstallConfigure) Interface {
	drs := datasource.NewIDataSource(cfg)
	return &SonarController{
		InstallConfigure: cfg,
		IService:         servicessonar.NewService(cfg, drs),
		IClient:          httpclient.NewIClient(),
		IDataSource:      drs,

		proc: proc.NewProc(),
	}
}

func sonarPipelineRunName(name string) string {
	name = strings.Replace(
		strings.Replace(strings.ToLower(
			name), "_", "-", -1), ".", "-", -1)
	name = fmt.Sprintf("%s-%s", name, "sonar")
	if len(name) > 62 {
		name = name[len(name)-62:]
	}
	return name
}
