package k8s

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
)

var ShardResources = NewResources()

type Resources struct {
	Data map[string]schema.GroupVersionResource
}

func NewResources() *Resources {
	return &Resources{
		Data: make(map[string]schema.GroupVersionResource),
	}
}

func (m *Resources) register(s string, resource schema.GroupVersionResource) {
	if _, exist := m.Data[s]; exist {
		return
	}
	m.Data[s] = resource
}

func (m *Resources) ranges(d dynamicinformer.DynamicSharedInformerFactory, stop <-chan struct{}) {
	for _, v := range m.Data {
		go d.ForResource(v).Informer().Run(stop)
	}
}

func init() {
	//ShardResources.register("", schema.GroupVersionResource{})
}
