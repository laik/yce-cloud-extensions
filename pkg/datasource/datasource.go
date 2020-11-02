package datasource

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type IDataSource interface {
	List(namespace, resource, flag string, pos, size int64, selector interface{}) (*unstructured.UnstructuredList, error)
	Get(namespace, resource, name string, subresources ...string) (runtime.Object, error)
	Apply(namespace, resource, name string, obj *unstructured.Unstructured) (*unstructured.Unstructured, bool, error)
	Delete(namespace, resource, name string) error
	Watch(namespace string, resource, resourceVersion string, timeoutSeconds int64, selector interface{}) (<-chan watch.Event, error)
}

func NewIDataSource() IDataSource {
	return nil
}
