package ci

import (
	"fmt"
	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"github.com/laik/yce-cloud-extensions/pkg/datasource"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	"github.com/laik/yce-cloud-extensions/pkg/services"
	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

var _ services.IService = &Service{}

type Service struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
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
				fmt.Printf("pipeline run channel recv handle object (%v) error (%s)\n", pipelineRun.Object, err)
				return
			}
		case ci, ok := <-ciChan:
			if !ok {
				fmt.Printf("ci channel closed\n")
				return
			}
			ciObj, ok := ci.Object.(*v1.CI)
			if !ok {
				fmt.Printf("ci channel recv object can't not convert to ci object (%v)\n", ci.Object)
				return
			}
			if err := c.reconcileCI(ciObj); err != nil {
				fmt.Printf("ci channel recv handle object (%v) error (%s)\n", ci.Object, err)
				return
			}
		}
	}
}

func (c *Service) reconcilePipelineRun(runtimeObject runtime.Object) error {
	//
	return nil
}

// Generator Tekton Task/Pipeline/PipelineResource/PipelineRun/Config...
func (c *Service) reconcileCI(ci *v1.CI) error {
	projectName, err := tools.ExtractProject(*ci.Spec.GitURL)
	if err != nil {
		return fmt.Errorf("illegal project name extract from git url (%s)", *ci.Spec.GitURL)
	}

	// Check Secret Config install
	obj, err := c.checkAndRecreateConfig()
	if err != nil {
		return err
	}

	prName := pipelineRunName(projectName, *ci.Spec.Branch)
	pipelineResourceName := fmt.Sprintf(pipelineResourceNameModel, prName)
	// first create pipelineResource with pipelineRun same name
	obj, err = c.checkAndRecreatePipelineResource(pipelineResourceName, *ci.Spec.GitURL, *ci.Spec.Branch)
	if err != nil {
		return err
	}

	// second create graph
	pipelineRunGraphName := fmt.Sprintf("%s-%d", pipelineGraphName, time.Now().Unix())
	obj, err = c.checkAndRecreateGraph(pipelineRunGraphName)
	if err != nil {
		return err
	}

	// pipelineRun reconcile
	obj, err = c.checkAndRecreatePipelineRun(prName, pipelineRunGraphName)
	if err != nil {
		return err
	}

	_ = obj
	return nil
}

func (c *Service) checkAndRecreateConfig() (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.TektonConfig, TektonConfigName)
	configParams := &parameter{
		Namespace:    common.YceCloudExtensionsOps,
		Name:         TektonConfigName,
		ConfigGitUrl: configGitUrl,
		Username:     configGitUser,
		Password:     configGitPassword,
	}
	defaultConfig, err := render(configParams, configTpl)
	if err != nil {
		return nil, err
	}
	if !errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, TektonConfigName, obj)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if !tools.CompareSpecByUnstructured(defaultConfig, obj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonConfig, TektonConfigName, defaultConfig)
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

	if !tools.ContainStringItem(newValue, TektonConfigName) {
		newValue = append(newValue, TektonConfigName)
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
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.Task, taskName)
	taskParams := &parameter{
		Namespace: common.YceCloudExtensionsOps,
		Name:      taskName,
	}
	defaultTask, err := render(taskParams, taskTpl)
	if err != nil {
		return nil, err
	}
	if !errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Task, taskName, obj)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if !tools.CompareSpecByUnstructured(defaultTask, obj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Task, taskName, defaultTask)
		if err != nil {
			return nil, err
		}
	}
	return obj, nil
}

func (c *Service) checkAndRecreatePipeline(projectName, ProjectVersion string) (*unstructured.Unstructured, error) {
	getObj, err := c.Get(common.YceCloudExtensionsOps, k8s.Pipeline, taskName)
	pipelineParams := &parameter{
		Namespace:      common.YceCloudExtensionsOps,
		Name:           pipelineName,
		ProjectName:    projectName,
		ProjectVersion: ProjectVersion,
		BuildToolImage: buildToolImage,
		DestRepoUrl:    destRepoUrl,
		CacheRepoUrl:   cacheRepoUrl,
	}
	obj, err := render(pipelineParams, pipelineTpl)
	if err != nil {
		return nil, err
	}

	if errors.IsNotFound(err) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Pipeline, pipelineName, obj)
		if err != nil {
			return nil, err
		}
		return obj, nil
	}

	if err != nil {
		return nil, err
	}

	if !tools.CompareSpecByUnstructured(obj, getObj) {
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.Pipeline, pipelineName, obj)
		if err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func (c *Service) checkAndRecreatePipelineRun(name, pipelineRunGraphName string) (*unstructured.Unstructured, error) {
	obj, err := c.Get(common.YceCloudExtensionsOps, k8s.PipelineRun, name)
	if errors.IsNotFound(err) {
		// create pipelineRun
		pipelineRunParams := &parameter{
			Namespace:            common.YceCloudExtensions,
			Name:                 name,
			PipelineName:         pipelineName,
			PipelineGraph:        pipelineGraphName,
			PipelineRunGraph:     pipelineRunGraphName,
			PipelineResourceName: pipelineResourceNameModel,
		}
		obj, err = render(pipelineRunParams, pipelineRunTpl)
		if err != nil {
			return nil, err
		}
		obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.PipelineRun, name, obj)
		if err != nil {
			return nil, err
		}
	}

	// Check the object status if failed callback ..
	//obj.Object

	return obj, err
}

func (c *Service) checkAndRecreateGraph(name string) (*unstructured.Unstructured, error) {
	graphParams := &parameter{
		Namespace: common.YceCloudExtensionsOps,
		Name:      name,
	}
	obj, err := render(graphParams, graphTpl)
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
	pipelineResourceParams := &parameter{
		Namespace: common.YceCloudExtensionsOps,
		Name:      pipelineResourceNameModel,
		GitUrl:    gitUrl,
		Branch:    branch,
	}
	obj, err := render(pipelineResourceParams, pipelineResourceTpl)
	if err != nil {
		return nil, err
	}
	obj, _, err = c.Apply(common.YceCloudExtensionsOps, k8s.TektonGraph, name, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func pipelineRunName(project, branch string) string {
	return fmt.Sprintf("%s-%s", project, branch)
}
