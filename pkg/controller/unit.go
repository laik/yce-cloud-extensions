package controller

import (
	"bytes"
	"context"
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
	servicesunit "github.com/laik/yce-cloud-extensions/pkg/services/unit"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	httpclient "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	"github.com/tidwall/gjson"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

type UnitController struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
	services.IService
	lastVersion string

	proc *proc.Proc
}

func (s *UnitController) response2echoer(data map[string]interface{}) error {
	request := s.Post(common.EchoerAddr)
	for k, v := range data {
		request.Params(k, v)
	}
	if err := request.Do(); err != nil {
		return err
	}
	return nil
}

func (s *UnitController) getLog(unit *v1.Unit) (string, error) {

	pipelineRun, err := s.Get(common.YceCloudExtensionsOps, k8s.PipelineRun, unit.Name)
	if err != nil {
		return "", err
	}
	pipelineRunBytes, err := pipelineRun.MarshalJSON()
	if err != nil {
		return "", err
	}
	podName := gjson.Get(string(pipelineRunBytes), "status.taskRuns.*.status.podName").String()
	req := s.Clientset.CoreV1().Pods(common.YceCloudExtensionsOps).GetLogs(podName, &corev1.PodLogOptions{
		Container: "step-step2",
	})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *UnitController) reconcile(unit *v1.Unit) error {
	if !unit.Spec.Done || len(unit.Spec.AckStates) == 0 {
		return nil
	}

	bufString, err := s.getLog(unit)
	resp := &resource.UnitResponse{
		FlowId:   *unit.Spec.FlowId,
		StepName: *unit.Spec.StepName,
		AckState: unit.Spec.AckStates[0],
		UUID:     *unit.Spec.UUID,
		Done:     unit.Spec.Done,
		Data:     bufString,
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

func (s *UnitController) recv(stop <-chan struct{}, errC chan<- error) {
	list, err := s.List(common.YceCloudExtensionsOps, k8s.UNIT, "", 0, 0, nil)
	if err != nil {
		errC <- err
		return
	}

	for _, item := range list.Items {
		value := item
		unit := &v1.Unit{}
		if err := tools.UnstructuredObjectToInstanceObj(&value, unit); err != nil {
			fmt.Printf("%s UnstructuredObjectToInstanceObj error (%s)", common.ERROR, err)
			continue
		}
		if err := s.reconcile(unit); err != nil {
			fmt.Printf("%s handle unit error (%s)\n", common.ERROR, err)
			continue
		}
	}

	unitList := &v1.UnitList{}
	if err := tools.UnstructuredListObjectToInstanceObjectList(list, unitList); err != nil {
		errC <- fmt.Errorf("UnstructuredListObjectToInstanceObjectList error (%s) (%v)", err, list)
		return
	}

	eventChan, err := s.Watch(common.YceCloudExtensionsOps, k8s.UNIT, unitList.GetResourceVersion(), 0, nil)
	if err != nil {
		errC <- err
		return
	}

	fmt.Printf("%s unit controller start watch unit channel.....\n", common.INFO)

	for {
		select {
		case <-stop:
			fmt.Printf("%s unit controller stop\n", common.INFO)
			return

		case item, ok := <-eventChan:
			if !ok {
				fmt.Printf("%s unit controller watch stone resource channel stop\n", common.ERROR)
				errC <- fmt.Errorf("controller watch unit channel closed")
				return
			}

			unit := &v1.Unit{}
			err := tools.RuntimeObjectToInstance(item.Object, unit)
			if err != nil {
				fmt.Printf("%s RuntimeObjectToInstance error (%s)\n", common.WARN, err)
				continue
			}

			if err := s.reconcile(unit); err != nil {
				fmt.Printf("%s unit controller handle error (%s)\n", common.ERROR, err)
				continue
			}

			s.lastVersion = unit.GetResourceVersion()
		}
	}
}

func (s *UnitController) checkAndReconcileUnit(name string) error {
	obj, err := s.Get(common.YceCloudExtensionsOps, k8s.UNIT, name)
	if err != nil {
		return err
	}
	unit := &v1.Unit{}
	if err := tools.UnstructuredObjectToInstanceObj(obj, unit); err != nil {
		return err
	}
	if unit.Spec.Done == false {
		unit.Spec.AckStates = append(unit.Spec.AckStates, v1.FailState)
		unit.Spec.Done = true
		unitUnstructured, err := tools.InstanceToUnstructured(unit)
		if err != nil {
			return err
		}
		if _, _, err := s.Apply(common.YceCloudExtensionsOps, k8s.UNIT, name, unitUnstructured, false); err != nil {
			return err
		}
	}
	return nil
}

func (s *UnitController) Run(addr string) error {
	route := gin.New()
	route.Use(gin.Logger())

	route.POST("/", func(g *gin.Context) {
		// 接收到 echoer post 的请求数据
		rawData, err := g.GetRawData()
		if err != nil {
			requestErr(g, err)
			return
		}
		request := &resource.RequestUnit{}
		if err := json.Unmarshal(rawData, request); err != nil {
			requestErr(g, err)
			return
		}
		// 构造UNIT的参数
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

		name = pipelineRunName(name)

		err = s.checkAndReconcileUnit(name)
		if err != nil {
			fmt.Printf("check last unit error%s", err)
		}

		// 构造一个UNIT的结构
		unit := &v1.Unit{
			TypeMeta: metav1.TypeMeta{
				Kind:       "UNIT",
				APIVersion: "yamecloud.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: common.YceCloudExtensionsOps,
			},
			Spec: v1.UnitSpec{
				GitURL:   &request.GitUrl,
				Branch:   &request.Branch,
				Language: &request.Language,
				Build:    &request.Build,
				Version:  &request.Version,
				Command:  &request.Command,

				FlowId:    &request.FlowId,
				StepName:  &request.StepName,
				AckStates: request.AckStates,
				UUID:      &request.UUID,
				Done:      false,
			},
		}
		// 转换成unstructured 类型
		unstructured, err := tools.InstanceToUnstructured(unit)
		if err != nil {
			requestErr(g, err)
			return
		}
		// 写入CRD配置
		obj, _, err := s.Apply(common.YceCloudExtensionsOps, k8s.UNIT, name, unstructured, true)
		if err != nil {
			internalApplyErr(g, err)
			fmt.Printf("unit controller apply (%s) error (%s)\n", name, err)
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

func NewUnitController(cfg *configure.InstallConfigure) Interface {
	drs := datasource.NewIDataSource(cfg)
	return &UnitController{
		InstallConfigure: cfg,
		IService:         servicesunit.NewService(cfg, drs),
		IClient:          httpclient.NewIClient(),
		IDataSource:      drs,

		proc: proc.NewProc(),
	}
}

func pipelineRunName(name string) string {
	name = strings.Replace(
		strings.Replace(strings.ToLower(
			name), "_", "-", -1), ".", "-", -1)
	name = fmt.Sprintf("%s-%s", name, "unit")
	if len(name) > 62 {
		name = name[len(name)-62:]
	}
	return name
}
