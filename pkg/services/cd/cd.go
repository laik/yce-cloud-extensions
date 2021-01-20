package cd

import (
	"encoding/json"
	"fmt"
	"github.com/laik/yce-cloud-extensions/pkg/utils/dict"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"

	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"github.com/laik/yce-cloud-extensions/pkg/datasource"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	"github.com/laik/yce-cloud-extensions/pkg/services"
	"github.com/laik/yce-cloud-extensions/pkg/utils/tools"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

var _ services.IService = &Service{}

type Service struct {
	*configure.InstallConfigure
	datasource.IDataSource
	lastCDVersion    string
	lastStoneVersion string
}

func (c *Service) Start(stop <-chan struct{}, errC chan<- error) {
	cdChan, err := c.Watch(common.YceCloudExtensions, k8s.CD, c.lastCDVersion, 0, nil)
	if err != nil {
		fmt.Printf("%s watch cd resource error (%s)\n", common.ERROR, err)
		errC <- err
	}
	stoneChan, err := c.Watch("", k8s.Stone, c.lastStoneVersion, 0, "yce-cloud-extensions")
	if err != nil {
		fmt.Printf("%s watch cd resource error (%s)\n", common.ERROR, err)
		errC <- err
	}

	fmt.Printf("%s service cd start watch ci channel and pipeline run channel\n", common.INFO)
	for {
		select {
		case <-stop:
			fmt.Printf("%s service cd get stop order\n", common.INFO)
			return
		case stoneEvent, ok := <-stoneChan:
			if !ok {
				fmt.Printf("%s service cd watch stone resource channel stop\n", common.ERROR)
				errC <- fmt.Errorf("watch stone channel closed")
				return
			}
			if stoneEvent.Type == watch.Deleted {
				continue
			}

			stone := stoneEvent.Object
			if err := c.reconcileStone(stone); err != nil {
				fmt.Printf("%s service cd reconcile stone error(%s)\n", common.ERROR, err)
			}

			// record watch version
			result, err := tools.GetObjectValue(stone, "metadata.resourceVersion")
			if err != nil {
				fmt.Printf("%s cd service watch stone resource version not found\n", common.ERROR)
				continue
			}

			c.lastStoneVersion = result.String()

		case item, ok := <-cdChan:
			if !ok {
				fmt.Printf("%s service cd watch cd resource channel stop\n", common.ERROR)
				errC <- fmt.Errorf("service cd watch cd channel closed")
			}

			cd := &v1.CD{}
			if err := tools.RuntimeObjectToInstance(item.Object, cd); err != nil {
				fmt.Printf("%s service cd convert cd (%s) resource error (%s)\n", common.ERROR, cd.GetName(), err)
				continue
			}

			if err := c.reconcileCD(cd); err != nil {
				fmt.Printf("%s service cd reconcile (%s) handle error (%s)\n", common.ERROR, cd.GetName(), err)
				continue
			}
			// record watch version
			c.lastCDVersion = cd.GetResourceVersion()
		}
	}
}

func labelsToQuery(data map[string]string) string {
	result := make([]string, 0)
	for k, v := range data {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(result, ",")
}

func (c *Service) reconcileStone(stone runtime.Object) error {
	stoneBytes, err := json.Marshal(stone)
	if err != nil {
		return err
	}
	//labels := make(map[string]string)
	//if err := json.Unmarshal([]byte(gjson.Get(string(stoneBytes), "metadata.labels").String()), &labels); err != nil {
	//	return err
	//}
	//
	//expectedUpdatedReplicas := gjson.Get(string(stoneBytes), "status.replicas").Int()
	//expectedImages := make([]string, 0)
	//gjson.Get(string(stoneBytes), "spec.template.spec").ForEach(func(key, value gjson.Result) bool {
	//	c := &corev1.Container{}
	//	if err := json.Unmarshal([]byte(value.String()), c); err != nil {
	//		return false
	//	}
	//	expectedImages = append(expectedImages, c.Image)
	//	return true
	//})
	//
	//// need compare expectedImages
	//
	//unstructuredPods, err := c.List(common.YceCloudExtensionsOps, k8s.Pod, "", 0, 0, labelsToQuery(labels))
	//if err != nil {
	//	return err
	//}
	//if int64(len(unstructuredPods.Items)) != expectedUpdatedReplicas {
	//	fmt.Printf(
	//		"%s expected deploy or update replicas not match,waiting reconcile stone %s\n",
	//		common.INFO,
	//		gjson.Get(string(stoneBytes), "metadata.name").String(),
	//	)
	//	return nil
	//}

	//// Need stone condition state support
	//deployDone := true
	//for _, item := range unstructuredPods.Items {
	//	podBytes, err := json.Marshal(item)
	//	if err != nil {
	//		return err
	//	}
	//	if gjson.Get(string(podBytes), "status.phase").String() != "Running" {
	//		deployDone = false
	//	}
	//}
	//
	//// if the deploy task not done, wait next event
	//if !deployDone {
	//	return nil
	//}

	var name = ""
	gjson.Get(string(stoneBytes), "metadata.labels").ForEach(func(k, v gjson.Result) bool {
		if k.String() != "yce-cloud-extensions" {
			return true
		}
		name = v.String()
		return false
	})
	if name == "" {
		return nil
	}
	unstructuredCD, err := c.Get(common.YceCloudExtensions, k8s.CD, name)
	if err != nil {
		return nil
	}

	cd := &v1.CD{}
	if err := tools.UnstructuredObjectToInstanceObj(unstructuredCD, cd); err != nil {
		return err
	}
	if cd.Spec.Done {
		return nil
	}

	cd.Spec.Done = true
	cd.Spec.AckStates = []string{v1.SuccessState}

	newUnstructuredCD, err := tools.InstanceToUnstructured(cd)
	if err != nil {
		return err
	}

	if _, _, err := c.Apply(common.YceCloudExtensions, k8s.CD, name, newUnstructuredCD, false); err != nil {
		return err
	}

	return nil
}

func (c *Service) reconcileCD(cd *v1.CD) error {
	if cd.Spec.Done {
		return nil
	}
	unstructuredNamespace, err := c.Get("", k8s.Namespace, *cd.Spec.DeployNamespace)
	if err != nil {
		return fmt.Errorf("reconcile cd (%s) can't not get deploy namespace (%s) error (%s)",
			cd.Name,
			*cd.Spec.DeployNamespace,
			err,
		)
	}
	namespaceBytes, err := json.Marshal(unstructuredNamespace.Object)
	if err != nil {
		return err
	}
	resourceLimitContent := gjson.Get(string(namespaceBytes), `metadata.annotations.nuwa\.kubernetes\.io\/default_resource_limit`).String()
	if resourceLimitContent == "" {
		return fmt.Errorf("namespace (%s) not allow workload node for deploy (%s)",
			*cd.Spec.DeployNamespace,
			fmt.Sprintf(`{"namespace"":"%s","app"":"%s"}`, *cd.Spec.DeployNamespace, *cd.Spec.ServiceName),
		)
	}

	namespaceResourceLimitSlice := make(NamespaceResourceLimitSlice, 0)
	if err := json.Unmarshal([]byte(resourceLimitContent), &namespaceResourceLimitSlice); err != nil {
		return fmt.Errorf(
			"namespace (%s) not allow workload node because don't unmarshal content (%s)",
			*cd.Spec.DeployNamespace,
			resourceLimitContent,
		)
	}

	configVolumes := make([]v1.ConfigVolumes, 0)
	if cd.Spec.ArtifactInfo.ConfigVolumes == nil {
		cd.Spec.ArtifactInfo.ConfigVolumes = configVolumes
	} else {
		for idx, configVolume := range cd.Spec.ArtifactInfo.ConfigVolumes {
			pathStr := strings.Split(configVolume.MountPath, "/")
			if len(pathStr) < 2 {
				configVolume.SubPath = ""
				continue
			} else {
				cd.Spec.ArtifactInfo.ConfigVolumes[idx].SubPath = pathStr[len(pathStr)-1]
			}
		}
	}

	params := &params{
		CDName:         cd.GetName(),
		Namespace:      *cd.Spec.DeployNamespace,
		Name:           *cd.Spec.ServiceName,
		Image:          *cd.Spec.ServiceImage,
		CpuLimit:       *cd.Spec.CPULimit,
		MemoryLimit:    *cd.Spec.MEMLimit,
		CpuRequests:    *cd.Spec.CPURequests,
		MemoryRequests: *cd.Spec.MEMRequests,
		ConfigVolumes:  cd.Spec.ArtifactInfo.ConfigVolumes,
		Commands:       cd.Spec.ArtifactInfo.Command,
		Args:           cd.Spec.ArtifactInfo.Arguments,
		Environments:   cd.Spec.ArtifactInfo.Environments,
		ServicePorts:   cd.Spec.ArtifactInfo.ServicePorts,
		ServiceType:    "ClusterIP",
		Coordinates:    createResourceLimitStructs(namespaceResourceLimitSlice.GroupBy(), cd.Spec.Replicas),
		UUID:           fmt.Sprintf("%s-%s", *cd.Spec.DeployNamespace, *cd.Spec.ServiceName),
	}

	if len(cd.Spec.ArtifactInfo.ConfigVolumes) != 0 {

		configMap := make(map[string]interface{})
		dict.Set(configMap, "apiVersion", "v1")
		dict.Set(configMap, "metadata.name", *cd.Spec.ServiceName)

		dataValue := make(map[string]interface{})
		for _, configVolume := range cd.Spec.ArtifactInfo.ConfigVolumes {
			for _, cnItem := range configVolume.CMItems {
				dataValue[cnItem.VolumeName] = cnItem.VolumeData
			}
		}
		dict.Set(configMap, "data", dataValue)
		unstructuredConfigMap := &unstructured.Unstructured{Object: configMap}
		_, _, err = c.Apply(*cd.Spec.DeployNamespace, k8s.ConfigMap, *cd.Spec.ServiceName, unstructuredConfigMap, true)
		if err != nil {
			return fmt.Errorf("%s configMap apply error (%s)\n", common.ERROR, err)
		}
	}

	unstructuredStone, err := services.Render(params, stoneTpl)
	if err != nil {
		return fmt.Errorf("stone render error (%s)", err)
	}

	_, _, err = c.Apply(*cd.Spec.DeployNamespace, k8s.Stone, *cd.Spec.ServiceName, unstructuredStone, true)
	if err != nil {
		return fmt.Errorf("%s stone apply error (%s)\n", common.ERROR, err)
	}

	return nil
}

func NewCDService(cfg *configure.InstallConfigure, dsrc datasource.IDataSource) *Service {
	return &Service{
		InstallConfigure: cfg,
		IDataSource:      dsrc,
		lastStoneVersion: "0",
		lastCDVersion:    "0",
	}
}
