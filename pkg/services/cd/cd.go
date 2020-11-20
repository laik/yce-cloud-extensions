package cd

import (
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

var _ services.IService = &CDService{}

type CDService struct {
	*configure.InstallConfigure
	datasource.IDataSource
}

func (c *CDService) Start(stop <-chan struct{}) {
	cdChan, err := c.Watch(common.YceCloudExtensions, k8s.CD, "0", 0, nil)
	if err != nil {
		fmt.Printf("%s watch cd resource error (%s)\n", common.ERROR, err)
		return
	}
	stoneChan, err := c.Watch(common.YceCloudExtensions, k8s.Stone, "0", 0, nil)
	if err != nil {
		fmt.Printf("%s watch cd resource error (%s)\n", common.ERROR, err)
		return
	}
	for {
		select {
		case <-stop:
			fmt.Printf("%s cd service get stop order\n", common.INFO)
			return
		case stoneEvent, ok := <-stoneChan:
			if !ok {
				fmt.Printf("%s cd service watch stone resource channel stop\n", common.ERROR)
				return
			}
			if stoneEvent.Type == watch.Added || stoneEvent.Type == watch.Modified {
				if err := c.reconcileStone(stoneEvent.Object); err != nil {
					fmt.Printf("%s reconcile stone error(%s)\n", common.ERROR, err)
				}
			}
		case item, ok := <-cdChan:
			if !ok {
				fmt.Printf("%s cd service watch cd resource channel stop\n", common.ERROR)
				return
			}
			cd := &v1.CD{}
			if err := tools.RuntimeObjectToInstance(item.Object, cd); err != nil {
				fmt.Printf("%s cd service convert cd resource error(%s)\n", common.ERROR, err)
				continue
			}
			if err := c.reconcileCD(cd); err != nil {
				fmt.Printf("reconcile cd handle error (%s)\n", err)
				continue
			}
		}
	}
}

func (c *CDService) reconcileStone(stone runtime.Object) error {
	stoneBytes, err := json.Marshal(stone)
	if err != nil {
		return err
	}
	labels := make(map[string]string)
	if err := json.Unmarshal([]byte(gjson.Get(string(stoneBytes), "metadata.labels").String()), labels); err != nil {
		return err
	}

	return nil
}

func (c *CDService) reconcileCD(cd *v1.CD) error {
	unstructuredNamespace, err := c.Get("", k8s.Namespace, *cd.Spec.DeployNamespace)
	if err != nil {
		return err
	}
	namespaceBytes, err := json.Marshal(unstructuredNamespace.Object)
	if err != nil {
		return err
	}
	resourceLimitContent := gjson.Get(string(namespaceBytes), "metadata.annotations.nuwa.kubernetes.io/default_resource_limit").String()
	if resourceLimitContent == "" {
		return fmt.Errorf("namespace (%s) not allow workload node", *cd.Spec.DeployNamespace)
	}

	namespaceResourceLimitSlice := make(namespaceResourceLimitSlice, 0)
	if err := json.Unmarshal([]byte(resourceLimitContent), &namespaceResourceLimitSlice); err != nil {
		return fmt.Errorf(
			"namespace (%s) not allow workload node because don't unmarshal content (%s)",
			*cd.Spec.DeployNamespace,
			resourceLimitContent,
		)
	}

	resourceLimitStructsBytes, err := createResourceLimitStructs(namespaceResourceLimitSlice.GroupBy(), cd.Spec.Replicas)
	if err != nil {
		return err
	}

	params := &params{
		Namespace:      *cd.Spec.DeployNamespace,
		Name:           *cd.Spec.ServiceName,
		Image:          *cd.Spec.ServiceImage,
		CpuLimit:       *cd.Spec.CPULimit,
		MemoryLimit:    *cd.Spec.MEMRequests,
		CpuRequests:    *cd.Spec.CPURequests,
		MemoryRequests: *cd.Spec.MEMRequests,
		ServicePorts:   cd.Spec.ArtifactInfo.ServicePorts,
		ServiceType:    "ClusterIP",
		Coordinates:    string(resourceLimitStructsBytes),
		UUID:           fmt.Sprintf("%s-%s", *cd.Spec.DeployNamespace, *cd.Spec.ServiceName),
	}

	unstructuredStone, err := services.Render(params, stoneTpl)
	if err != nil {
		return err
	}

	obj, _, err := c.Apply(cd.Namespace, k8s.Stone, cd.Name, unstructuredStone)
	if err != nil {
		fmt.Printf("%s stone apply error (%s)", common.ERROR, err)
		return err
	}
	_ = obj

	return nil
}

func NewCDService(cfg *configure.InstallConfigure, dsrc datasource.IDataSource) *CDService {
	return &CDService{
		InstallConfigure: cfg,
		IDataSource:      dsrc,
	}
}
