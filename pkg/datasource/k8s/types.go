package k8s

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
)

type ResourceLister interface {
	Ranges(d dynamicinformer.DynamicSharedInformerFactory, stop <-chan struct{})
	GetGvr(string) (schema.GroupVersionResource, error)
}

const (
	// CI && CD YameCloudExtensions resources
	CI = "cis"
	CD = "cds"

	// Tekton resources
	Pipeline         = "pipelines"
	PipelineRun      = "pipelineruns"
	Task             = "tasks"
	TaskRun          = "taskruns"
	PipelineResource = "pipelineresources"

	// Extend Tekton Pipeline & PipelineRun resource Graph
	TektonGraph  = "tektongraphs"
	TektonConfig = "secrets"

	// Kubernetes
	ServiceAccount = "serviceaccounts"
)

type Resources struct {
	excluded []string

	Data map[string]schema.GroupVersionResource
}

func NewResources(executeRes []string) *Resources {
	rs := &Resources{
		excluded: executeRes,

		Data: make(map[string]schema.GroupVersionResource),
	}

	rsInit(rs)

	return rs
}

func (m *Resources) register(s string, resource schema.GroupVersionResource) {
	if _, exist := m.Data[s]; exist {
		return
	}
	m.Data[s] = resource
}

func (m *Resources) Ranges(d dynamicinformer.DynamicSharedInformerFactory, stop <-chan struct{}) {
	for _, v := range m.excluded {
		value := v
		delete(m.Data, value)
	}
	for _, v := range m.Data {
		value := v
		go d.ForResource(value).Informer().Run(stop)
	}
}

func (m *Resources) GetGvr(s string) (schema.GroupVersionResource, error) {
	item, exist := m.Data[s]
	if !exist {
		return schema.GroupVersionResource{}, fmt.Errorf("resource (%s) not exist", s)
	}
	return item, nil
}

func rsInit(rs *Resources) {
	rs.register(CI, schema.GroupVersionResource{Group: "yamecloud.io", Version: "v1", Resource: CI})
	rs.register(CD, schema.GroupVersionResource{Group: "yamecloud.io", Version: "v1", Resource: CD})

	// tekton.dev resource view
	rs.register(Pipeline, schema.GroupVersionResource{Group: "tekton.dev", Version: "v1alpha1", Resource: "pipelines"})
	rs.register(PipelineRun, schema.GroupVersionResource{Group: "tekton.dev", Version: "v1alpha1", Resource: "pipelineruns"})
	rs.register(Task, schema.GroupVersionResource{Group: "tekton.dev", Version: "v1alpha1", Resource: "tasks"})
	rs.register(TaskRun, schema.GroupVersionResource{Group: "tekton.dev", Version: "v1alpha1", Resource: "taskruns"})
	rs.register(PipelineResource, schema.GroupVersionResource{Group: "tekton.dev", Version: "v1alpha1", Resource: "pipelineresources"})

	// tekton graph
	rs.register(TektonGraph, schema.GroupVersionResource{Group: "fuxi.nip.io", Version: "v1", Resource: "tektongraphs"})
	rs.register(TektonConfig, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"})
	rs.register(ServiceAccount, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"})
}
