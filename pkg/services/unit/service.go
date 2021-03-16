package unit

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"github.com/laik/yce-cloud-extensions/pkg/datasource"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	"github.com/laik/yce-cloud-extensions/pkg/services"
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

var _ services.IService = &Service{}

type Service struct {
	*configure.InstallConfigure
	datasource.IDataSource
	lastPRVersion   string
	lastUNITVersion string
}

func NewService(cfg *configure.InstallConfigure, drs datasource.IDataSource) services.IService {
	return &Service{
		InstallConfigure: cfg,
		IDataSource:      drs,
		lastPRVersion:    "0",
		lastUNITVersion:  "0",
	}
}

func (c *Service) Start(stop <-chan struct{}, errC chan<- error) {
	pipelineRunChan, err := c.Watch(common.YceCloudExtensionsOps, k8s.PipelineRun, c.lastPRVersion, 0, nil)
	if err != nil {
		fmt.Printf("%s watch pipelineRun error (%s)\n", common.ERROR, err)
		errC <- err
	}

	unitChan, err := c.Watch(common.YceCloudExtensionsOps, k8s.UNIT, c.lastUNITVersion, 0, nil)
	if err != nil {
		fmt.Printf("%s watch pipelineRun error (%s)\n", common.ERROR, err)
		errC <- err
	}

	fmt.Printf("%s service unit start watch unit channel and pipeline run channel\n", common.INFO)

	for {
		select {
		case <-stop:
			fmt.Printf("%s service unit service get stop order\n", common.INFO)
			return
		case pipelineRunEvent, ok := <-pipelineRunChan:
			if !ok {
				fmt.Printf("%s service unit pipeline run channel closed\n", common.ERROR)
				errC <- fmt.Errorf("service unit watch pipeline run channel closed")
				return
			}
			if pipelineRunEvent.Type == watch.Deleted {
				continue
			}
			if err := c.reconcilePipelineRun(pipelineRunEvent.Object); err != nil {
				fmt.Printf("%s service unit watch pipeline run channel recv handle error (%s)\n", common.ERROR, err)
			}
			// record watch version
			result, err := tools.GetObjectValue(pipelineRunEvent.Object, "metadata.resourceVersion")
			if err != nil {
				fmt.Printf("%s service unit watch pipelinerun resource version not found\n", common.ERROR)
				continue
			}
			c.lastPRVersion = result.String()

		case unitEvent, ok := <-unitChan:
			if !ok {
				fmt.Printf("%s service unit channel closed\n", common.ERROR)
				errC <- fmt.Errorf("service watch unit channel closed")
				return
			}
			// ignore delete event
			if unitEvent.Type == watch.Deleted {
				continue
			}

			unitObj := &v1.Unit{}
			if err := tools.RuntimeObjectToInstance(unitEvent.Object, unitObj); err != nil {
				fmt.Printf("%s service unit channel recv object can't not convert to unit object (%s)\n", common.ERROR, err)
				continue
			}

			if err := c.reconcileUnit(unitObj); err != nil {
				fmt.Printf("%s service unit channel reconcil object (%s) error (%s)\n", common.ERROR, unitObj.GetName(), err)
			}
			c.lastUNITVersion = unitObj.GetResourceVersion()
		}
	}
}

type condition struct {
	LastTransitionTime string `json:"lastTransitionTime"`
	Message            string `json:"message"`
	Reason             string `json:"reason"`
	Status             string `json:"status"`
	Type               string `json:"type"`
}

func (c *Service) reconcilePipelineRun(runtimeObject runtime.Object) error {
	pipelineRunJSON, err := json.Marshal(runtimeObject)
	if err != nil {
		return err
	}
	pipelineRunJSONString := string(pipelineRunJSON)
	succeeded := "Succeeded"
	failed := "Failed"

	conditions := make([]*condition, 0)

	gjson.Get(pipelineRunJSONString, "status.conditions").ForEach(func(_, value gjson.Result) bool {
		c := &condition{}
		if err := json.Unmarshal([]byte(value.String()), c); err != nil {
			return false
		}
		conditions = append(conditions, c)
		return true
	})

	pipelineRunName := gjson.Get(pipelineRunJSONString, "metadata.name").String()
	if pipelineRunName == "" {
		return fmt.Errorf("not metadata.name on runtime object(%s)", runtimeObject)
	}

	if len(conditions) < 1 {
		return nil
	}

	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.UNIT, pipelineRunName)
	if err != nil {
		return fmt.Errorf("get unit %s", err)
	}
	unit := &v1.Unit{}
	if err := tools.UnstructuredObjectToInstanceObj(obj, unit); err != nil {
		return err
	}

	unit.Spec.AckStates = unit.Spec.AckStates[:0]
	switch {
	case conditions[0].Reason == succeeded && conditions[0].Status == "True" && conditions[0].Type == succeeded: // successed
		unit.Spec.Done = true
		unit.Spec.AckStates = append(unit.Spec.AckStates, v1.SuccessState)
	case conditions[0].Reason == failed && conditions[0].Status == "False" && conditions[0].Type == succeeded: // failed
		unit.Spec.Done = true
		unit.Spec.AckStates = append(unit.Spec.AckStates, v1.FailState)
	}

	ciUnstructured, err := tools.InstanceToUnstructured(unit)
	if err != nil {
		return err
	}
	if _, _, err := c.Apply(common.YceCloudExtensionsOps, k8s.UNIT, pipelineRunName, ciUnstructured, false); err != nil {
		return err
	}

	return nil
}

// Generator Tekton Task/Pipeline/PipelineResource/PipelineRun/Config...
func (c *Service) reconcileUnit(unit *v1.Unit) error {
	if unit.Spec.Done {
		return nil
	}
	projectName, err := tools.ExtractProject(*unit.Spec.GitURL)
	if err != nil {
		return fmt.Errorf("illegal project name extract from git url (%s)", *unit.Spec.GitURL)
	}

	// Check Secret Config install
	_, err = c.checkAndRecreateGitConfig()
	if err != nil {
		return fmt.Errorf("reconcile unit check and recreate config error (%s)", err)
	}
	// Check Secret Config install
	_, err = c.checkAndRecreateRegistryConfig()
	if err != nil {
		return fmt.Errorf("reconcile unit check and recreate config error (%s)", err)
	}

	prName := pipelineRunName(unit.ObjectMeta.Name)

	// first create pipelineResource with pipelineRun same name
	obj, err := c.checkAndRecreatePipelineResource(prName, *unit.Spec.GitURL, *unit.Spec.Branch)
	if err != nil {
		return err
	}

	// check and reconcile task normal
	if _, err = c.checkAndRecreateTask(); err != nil {
		return err
	}

	// check and reconcile pipeline graph
	_, err = c.checkAndRecreateGraph(services.UnitPipelineGraphName)
	if err != nil {
		return err
	}
	// check and reconcile pipeline
	if _, err := c.checkAndRecreatePipeline(); err != nil {
		return err
	}

	// check and reconcile pipelineRun graph
	pipelineRunGraphName := fmt.Sprintf("%s-%s", services.PipelineGraphName, prName)
	pipelineRunGraph, err := c.checkAndRecreateGraph(pipelineRunGraphName)
	if err != nil {
		return err
	}

	// check and reconcile pipelineRun
	obj, err = c.checkAndRecreatePipelineRun(
		prName,
		projectName,
		pipelineRunGraphName,
		prName,
		pipelineRunGraph,
		*unit.Spec.Language,
		*unit.Spec.Command,
	)
	if err != nil {
		return err
	}

	_ = obj
	return nil
}

func (c *Service) checkAndRecreateRegistryConfig() (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonDockerConfigName)

	configParams := params{
		Namespace:        common.YceCloudExtensionsOps,
		Name:             services.TektonDockerConfigName,
		RegistryRepoUrl:  services.ConfigRegistryUrl,
		RegistryUsername: base64.StdEncoding.EncodeToString([]byte(services.ConfigRegistryUserName)),
		RegistryPassword: base64.StdEncoding.EncodeToString([]byte(services.ConfigRegistryPassword)),
	}
	defaultConfig, err := services.Render(configParams, configRegistryTpl)
	if err != nil {
		return nil, err
	}
	if !errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonDockerConfigName, defaultConfig, false)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if !tools.CompareSpecByUnstructured(defaultConfig, obj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonDockerConfigName, defaultConfig, false)
		if err != nil {
			return nil, err
		}
	}

	serverAccount, err := c.Get(common.YceCloudExtensionsOps, k8s.ServiceAccount, "default")
	if err != nil {
		return nil, err
	}

	serviceAccountBytes, err := serverAccount.MarshalJSON()
	if err != nil {
		return nil, err
	}
	secretsPath := "secrets"

	var newValue = make([]string, 0)
	gjson.Get(string(serviceAccountBytes), secretsPath).
		ForEach(
			func(_, v gjson.Result) bool {
				newValue = append(newValue, v.String())
				return true
			})

	if len(newValue) < 1 {
		return nil, fmt.Errorf("get secrets not value")
	}

	if !tools.ContainStringItem(newValue, services.TektonDockerConfigName) {
		newValue = append(newValue, services.TektonDockerConfigName)
		newServiceAccountString, err := sjson.Set(string(serviceAccountBytes), secretsPath, newValue)
		if err != nil {
			return nil, err
		}
		serviceAccount := &unstructured.Unstructured{}
		if err := serviceAccount.UnmarshalJSON([]byte(newServiceAccountString)); err != nil {
			return nil, err
		}
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.ServiceAccount, serverAccount.GetName(), serviceAccount, false)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	return obj, nil
}

func (c *Service) checkAndRecreateGitConfig() (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonGitConfigName)

	configParams := params{
		Namespace:    common.YceCloudExtensionsOps,
		Name:         services.TektonGitConfigName,
		ConfigGitUrl: services.ConfigGitUrl,
		GitUsername:  base64.StdEncoding.EncodeToString([]byte(services.ConfigGitUser)),
		GitPassword:  base64.StdEncoding.EncodeToString([]byte(services.ConfigGitPassword)),
	}
	defaultConfig, err := services.Render(configParams, configGitTpl)
	if err != nil {
		return nil, err
	}
	if !errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonGitConfigName, defaultConfig, false)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if !tools.CompareSpecByUnstructured(defaultConfig, obj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonGitConfigName, defaultConfig, false)
		if err != nil {
			return nil, err
		}
	}

	serverAccount, err := c.Get(common.YceCloudExtensionsOps, k8s.ServiceAccount, "default")
	if err != nil {
		return nil, err
	}

	serviceAccountBytes, err := serverAccount.MarshalJSON()
	if err != nil {
		return nil, err
	}
	secretsPath := "secrets"

	var newValue = make([]string, 0)
	gjson.Get(string(serviceAccountBytes), secretsPath).
		ForEach(
			func(_, v gjson.Result) bool {
				newValue = append(newValue, v.String())
				return true
			})

	if len(newValue) < 1 {
		return nil, fmt.Errorf("get secrets not value")
	}

	if !tools.ContainStringItem(newValue, services.TektonGitConfigName) {
		newValue = append(newValue, services.TektonGitConfigName)
		newServiceAccountString, err := sjson.Set(string(serviceAccountBytes), secretsPath, newValue)
		if err != nil {
			return nil, err
		}
		serviceAccount := &unstructured.Unstructured{}
		if err := serviceAccount.UnmarshalJSON([]byte(newServiceAccountString)); err != nil {
			return nil, err
		}
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.ServiceAccount, serverAccount.GetName(), serviceAccount, false)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	return obj, nil
}

func (c *Service) checkAndRecreateTask() (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.Task, services.UnitTaskName)
	taskParams := params{
		Namespace: common.YceCloudExtensionsOps,
		Name:      services.UnitTaskName,
	}
	defaultTask, err := services.Render(taskParams, taskTpl)
	if err != nil {
		return nil, err
	}
	if !errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Task, services.UnitTaskName, defaultTask, false)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if !tools.CompareSpecByUnstructured(defaultTask, obj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Task, services.UnitTaskName, defaultTask, false)
		if err != nil {
			return nil, err
		}
	}
	return obj, nil
}

func (c *Service) checkAndRecreatePipeline() (*unstructured.Unstructured, error) {
	getObj, err := c.Get(common.YceCloudExtensionsOps, k8s.Pipeline, services.UnitTaskName)
	pipelineParams := params{
		Namespace:     common.YceCloudExtensionsOps,
		Name:          services.UnitPipelineName,
		PipelineGraph: services.UnitPipelineGraphName,
		TaskName:      services.UnitTaskName,
	}
	obj, err := services.Render(pipelineParams, pipelineTpl)
	if err != nil {
		return nil, err
	}

	if errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Pipeline, services.UnitPipelineName, obj, false)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if err != nil {
		return nil, err
	}

	if !tools.CompareSpecByUnstructured(obj, getObj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Pipeline, services.UnitPipelineName, obj, false)
		if err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func (c *Service) checkAndRecreatePipelineRun(
	name,
	projectName,
	pipelineRunGraphName,
	pipelineResourceName string,
	pipelineRunGraph *unstructured.Unstructured,
	codeType,
	command string,

) (*unstructured.Unstructured, error) {
	if codeType == "" {
		codeType = "none"
	}

	pipelineRunParams := params{
		Namespace:            common.YceCloudExtensionsOps,
		Name:                 name,
		PipelineName:         services.UnitPipelineName,
		PipelineGraph:        services.UnitPipelineGraphName,
		PipelineRunGraph:     pipelineRunGraphName,
		PipelineResourceName: pipelineResourceName,
		ProjectName:          projectName,
		BuildToolImage:       services.BuildToolImage,
		CacheRepoUrl:         services.CacheRepoUrl,
		CodeType:             codeType,
		Command:              command,
	}
	defaultObj, err := services.Render(pipelineRunParams, pipelineRunTpl)
	if err != nil {
		return nil, err
	}

	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.PipelineRun, name)
	if err != nil {
		if errors.IsNotFound(err) {
			// create pipelineRun
			obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.PipelineRun, name, defaultObj, false)
			if err != nil {
				return nil, err
			}
			goto OWNER_REF
		}
		return nil, err
	}

	err = c.Delete(common.YceCloudExtensionsOps, k8s.PipelineRun, name)
	if err != nil {
		return nil, err
	}
	obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.PipelineRun, name, defaultObj, false)
	if err != nil {
		return nil, err
	}
	pipelineRunGraph, err = c.checkAndRecreateGraph(pipelineRunGraph.GetName())
	if err != nil {
		return nil, err
	}

OWNER_REF:
	// reset graph owner
	pipelineRunGraphBytes, err := pipelineRunGraph.MarshalJSON()
	if err != nil {
		return nil, err
	}

	pipelineRunGraphObj, err := tools.SetObjectOwner(pipelineRunGraphBytes, obj.GetAPIVersion(), obj.GetKind(), obj.GetName(), string(obj.GetUID()))
	if err != nil {
		return nil, err
	}

	pipelineRunGraphObj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonGraph, pipelineRunGraphName, pipelineRunGraphObj, false)
	if err != nil {
		return nil, err
	}

	return obj, err
}

func (c *Service) checkAndRecreateGraph(name string) (*unstructured.Unstructured, error) {
	graphParams := params{
		Namespace: common.YceCloudExtensionsOps,
		Name:      name,
	}
	obj, err := services.Render(graphParams, graphTpl)
	if err != nil {
		return nil, err
	}
	obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonGraph, name, obj, false)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *Service) checkAndRecreatePipelineResource(name, gitUrl, branch string) (*unstructured.Unstructured, error) {
	pipelineResourceParams := params{
		Namespace: common.YceCloudExtensionsOps,
		Name:      name,
		GitUrl:    gitUrl,
		Branch:    branch,
	}
	obj, err := services.Render(pipelineResourceParams, pipelineResourceTpl)
	if err != nil {
		return nil, err
	}
	obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.PipelineResource, name, obj, false)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func pipelineRunName(name string) string {
	return strings.Replace(
		strings.Replace(strings.ToLower(
			name), "_", "-", -1), ".", "-", -1)
}
