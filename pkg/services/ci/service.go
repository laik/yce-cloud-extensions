package ci

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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
)

var _ services.IService = &Service{}

type Service struct {
	*configure.InstallConfigure
	datasource.IDataSource
}

func NewService(cfg *configure.InstallConfigure, drs datasource.IDataSource) services.IService {
	return &Service{
		InstallConfigure: cfg,
		IDataSource:      drs,
	}
}

func (c *Service) Start(stop <-chan struct{}) {
	pipelineRunChan, err := c.Watch(common.YceCloudExtensionsOps, k8s.PipelineRun, "0", 0, nil)
	if err != nil {
		fmt.Printf("watch pipelineRun error (%s)\n", err)
	}

	ciChan, err := c.Watch(common.YceCloudExtensionsOps, k8s.CI, "0", 0, nil)
	if err != nil {
		fmt.Printf("watch ciChan error (%s)\n", err)
	}
	for {
		select {
		case <-stop:
			return
		case pipelineRun, ok := <-pipelineRunChan:
			if !ok {
				fmt.Printf("pipeline run channel closed\n")
				return
			}
			if err := c.reconcilePipelineRun(pipelineRun.Object); err != nil {
				fmt.Printf("%s pipeline run channel recv handle object (%v) error (%s)\n", common.ERROR, pipelineRun.Object, err)
			}
		case ci, ok := <-ciChan:
			if !ok {
				fmt.Printf("ci channel closed\n")
				return
			}
			ciObj := &v1.CI{}
			if err := tools.RuntimeObjectToInstance(ci.Object, ciObj); err != nil {
				fmt.Printf("%s ci channel recv object can't not convert to ci object (%s)\n", common.ERROR, err)
			}
			if err := c.reconcileCI(ciObj); err != nil {
				fmt.Printf("%s ci channel reconcil error (%s)\n", common.ERROR, err)
			}
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
		fmt.Printf("%s reconcile pipelinerun (%s) not status.condistions real data (%s) \n",
			common.INFO,
			pipelineRunName,
			gjson.Get(pipelineRunJSONString, "status"),
		)
		return nil
	}

	obj, err := c.Get(common.YceCloudExtensionsOps, pipelineRunName, k8s.CI)
	if err != nil {
		return err
	}
	ci := &v1.CI{}
	if err := tools.UnstructuredObjectToInstanceObj(obj, ci); err != nil {
		return err
	}

	switch {
	case conditions[0].Reason == succeeded && conditions[0].Status == "True" && conditions[0].Type == succeeded: // successed
		ci.Spec.Done = true
		ci.Spec.AckStates = append(ci.Spec.AckStates, v1.SuccessState)
	case conditions[0].Reason == failed && conditions[0].Status == "False" && conditions[0].Type == succeeded: // failed
		ci.Spec.Done = true
		ci.Spec.AckStates = append(ci.Spec.AckStates, v1.FailState)
	}

	ciUnstructured, err := tools.InstanceToUnstructured(ci)
	if err != nil {
		return err
	}
	if _, _, err := c.Apply(common.YceCloudExtensionsOps, k8s.CI, pipelineRunName, ciUnstructured); err != nil {
		return err
	}

	return nil
}

// Generator Tekton Task/Pipeline/PipelineResource/PipelineRun/Config...
func (c *Service) reconcileCI(ci *v1.CI) error {
	projectName, err := tools.ExtractProject(*ci.Spec.GitURL)
	if err != nil {
		return fmt.Errorf("illegal project name extract from git url (%s)", *ci.Spec.GitURL)
	}

	// Check Secret Config install
	_, err = c.checkAndRecreateGitConfig()
	if err != nil {
		return fmt.Errorf("reconcile ci check and recreate config error (%s)", err)
	}
	// Check Secret Config install
	_, err = c.checkAndRecreateRegistryConfig()
	if err != nil {
		return fmt.Errorf("reconcile ci check and recreate config error (%s)", err)
	}

	prName := pipelineRunName(projectName, *ci.Spec.Branch)
	// first create pipelineResource with pipelineRun same name
	obj, err := c.checkAndRecreatePipelineResource(prName, *ci.Spec.GitURL, *ci.Spec.Branch)
	if err != nil {
		return err
	}

	// check and reconcile task normal
	if _, err = c.checkAndRecreateTask(); err != nil {
		return err
	}

	// check and reconcile pipeline graph
	_, err = c.checkAndRecreateGraph(services.PipelineGraphName)
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
	obj, err = c.checkAndRecreatePipelineRun(prName, projectName, *ci.Spec.CommitID, pipelineRunGraphName, prName, pipelineRunGraph)
	if err != nil {
		return err
	}

	_ = obj
	return nil
}

func (c *Service) checkAndRecreateRegistryConfig() (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonDockerConfigName)

	configParams := &services.Parameter{
		Namespace:        common.YceCloudExtensionsOps,
		Name:             services.TektonDockerConfigName,
		RegistryRepoUrl:  services.ConfigRegistryUrl,
		RegistryUsername: base64.StdEncoding.EncodeToString([]byte(services.ConfigRegistryUrl)),
		RegistryPassword: base64.StdEncoding.EncodeToString([]byte(services.ConfigRegistryPassword)),
	}
	defaultConfig, err := services.Render(configParams, configGitTpl)
	if err != nil {
		return nil, err
	}
	if !errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonDockerConfigName, defaultConfig)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if !tools.CompareSpecByUnstructured(defaultConfig, obj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonDockerConfigName, defaultConfig)
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
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.ServiceAccount, serverAccount.GetName(), serviceAccount)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	return obj, nil
}

func (c *Service) checkAndRecreateGitConfig() (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonGitConfigName)

	configParams := &services.Parameter{
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
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonGitConfigName, defaultConfig)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if !tools.CompareSpecByUnstructured(defaultConfig, obj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, services.TektonGitConfigName, defaultConfig)
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
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.ServiceAccount, serverAccount.GetName(), serviceAccount)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	return obj, nil
}

func (c *Service) checkAndRecreateTask() (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.Task, services.TaskName)
	taskParams := &services.Parameter{
		Namespace: common.YceCloudExtensionsOps,
		Name:      services.TaskName,
	}
	defaultTask, err := services.Render(taskParams, taskTpl)
	if err != nil {
		return nil, err
	}
	if !errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Task, services.TaskName, defaultTask)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if !tools.CompareSpecByUnstructured(defaultTask, obj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Task, services.TaskName, defaultTask)
		if err != nil {
			return nil, err
		}
	}
	return obj, nil
}

func (c *Service) checkAndRecreatePipeline() (*unstructured.Unstructured, error) {
	getObj, err := c.Get(common.YceCloudExtensionsOps, k8s.Pipeline, services.TaskName)
	pipelineParams := &services.Parameter{
		Namespace:     common.YceCloudExtensionsOps,
		Name:          services.PipelineName,
		PipelineGraph: services.PipelineGraphName,
		TaskName:      services.TaskName,
	}
	obj, err := services.Render(pipelineParams, pipelineTpl)
	if err != nil {
		return nil, err
	}

	if errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Pipeline, services.PipelineName, obj)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if err != nil {
		return nil, err
	}

	if !tools.CompareSpecByUnstructured(obj, getObj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Pipeline, services.PipelineName, obj)
		if err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func (c *Service) checkAndRecreatePipelineRun(
	name,
	projectName,
	projectVersion,
	pipelineRunGraphName,
	pipelineResourceName string,
	pipelineRunGraph *unstructured.Unstructured,

) (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.PipelineRun, name)
	if errors.IsNotFound(err) {
		// create pipelineRun
		pipelineRunParams := &services.Parameter{
			Namespace:            common.YceCloudExtensionsOps,
			Name:                 name,
			PipelineName:         services.PipelineName,
			PipelineGraph:        services.PipelineGraphName,
			PipelineRunGraph:     pipelineRunGraphName,
			PipelineResourceName: pipelineResourceName,
			ProjectName:          projectName,
			ProjectVersion:       projectVersion,
			BuildToolImage:       services.BuildToolImage,
			DestRepoUrl:          services.DestRepoUrl,
			CacheRepoUrl:         services.CacheRepoUrl,
		}
		obj, err = services.Render(pipelineRunParams, pipelineRunTpl)
		if err != nil {
			return nil, err
		}
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.PipelineRun, name, obj)
		if err != nil {
			return nil, err
		}
		goto OWNER_REF
	}

	// other err
	if err != nil {
		return nil, err
	} else {
		clonePipelineRunObject, err := tools.CloneNewObject(obj)
		if err != nil {
			return nil, err
		}

		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.PipelineRun, name, clonePipelineRunObject)
		if err != nil {
			return nil, err
		}

		pipelineRunGraph, err = c.checkAndRecreateGraph(pipelineRunGraph.GetName())
		if err != nil {
			return nil, err
		}
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

	pipelineRunGraphObj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonGraph, pipelineRunGraphName, pipelineRunGraphObj)
	if err != nil {
		return nil, err
	}

	return obj, err
}

func (c *Service) checkAndRecreateGraph(name string) (*unstructured.Unstructured, error) {
	graphParams := &services.Parameter{
		Namespace: common.YceCloudExtensionsOps,
		Name:      name,
	}
	obj, err := services.Render(graphParams, graphTpl)
	if err != nil {
		return nil, err
	}
	obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonGraph, name, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *Service) checkAndRecreatePipelineResource(name, gitUrl, branch string) (*unstructured.Unstructured, error) {
	pipelineResourceParams := &services.Parameter{
		Namespace: common.YceCloudExtensionsOps,
		Name:      name,
		GitUrl:    gitUrl,
		Branch:    branch,
	}
	obj, err := services.Render(pipelineResourceParams, pipelineResourceTpl)
	if err != nil {
		return nil, err
	}
	obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.PipelineResource, name, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func pipelineRunName(project, branch string) string {
	return fmt.Sprintf("%s-%s", project, branch)
}
