package cd

import (
	"fmt"
	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
	"github.com/laik/yce-cloud-extensions/pkg/common"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"github.com/laik/yce-cloud-extensions/pkg/datasource"
	"github.com/laik/yce-cloud-extensions/pkg/datasource/k8s"
	"github.com/laik/yce-cloud-extensions/pkg/services"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	client "github.com/laik/yce-cloud-extensions/pkg/utils/http"
	nuwav1 "github.com/yametech/nuwa/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ services.IService = &CDService{}

type CDService struct {
	*configure.InstallConfigure
	datasource.IDataSource
	client.IClient
}

func (c *CDService) Start(stop <-chan struct{}) {
	cdChan, err := c.Watch(common.YceCloudExtensions, k8s.CD, "0", 0, nil)
	if err != nil {
		//
	}
	for {
		select {
		case <-stop:
			fmt.Printf("CD Start get stop order")
			return
		case item, ok := <-cdChan:
			if !ok {

			}
			cd := item.Object.(*v1.CD)
			if err := c.handle(cd); err != nil {
				fmt.Printf("cd handle error (%s) (%v)", err, cd)
				continue
			}
		}
	}
}

func (c *CDService) handle(cd *v1.CD) error {
	labels := map[string]string{
		"app":               cd.Name,
		"app-template-name": "",
	} //TODO

	servicesSpec := &corev1.ServiceSpec{
		Type: corev1.ServiceType("ClusterIP"),
	} //TODO

	ports := cd.Spec.ArtifactInfo.ServicePorts
	for _, item := range ports {
		this := corev1.ServicePort{
			Name:       item.Name,
			Protocol:   corev1.Protocol(item.Protocol),
			Port:       item.Port,
			TargetPort: intstr.Parse(item.TargetPort),
		}
		servicesSpec.Ports = append(servicesSpec.Ports, this)
	}

	podSpec := corev1.PodSpec{} //TODO

	var cgs = make([]nuwav1.CoordinatesGroup, 0) //TODO

	stone := &nuwav1.Stone{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Stone",
			APIVersion: "nuwa.nip.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cd.Name,
			Namespace: cd.Namespace,
			Labels:    labels,
		},
		Spec: nuwav1.StoneSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   cd.Name,
					Labels: labels,
				},
				Spec: podSpec,
			},
			Strategy:             "Release",
			Coordinates:          cgs,
			Service:              *servicesSpec,
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{},
		},
	}

	unstructuredStone, err := runtime.DefaultUnstructuredConverter.ToUnstructured(stone)
	if err != nil {
		fmt.Printf("runtime ToUnstructured error (%s) (%v)", err, unstructuredStone)
		return err
	}

	obj, _, err := c.Apply(cd.Namespace, k8s.CD, cd.Name, &unstructured.Unstructured{Object: unstructuredStone})
	if err != nil {
		fmt.Printf("CD Apply error (%s) (%v)", err, obj)
		return err
	}
	return nil
}

func init() {

}
