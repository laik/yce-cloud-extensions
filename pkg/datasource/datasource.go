package datasource

import (
	"context"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
)

var _ IDataSource = &IDataSourceImpl{}

type IDataSource interface {
	List(namespace, resource, flag string, pos, size int64, selector interface{}) (*unstructured.UnstructuredList, error)
	Get(namespace, resource, name string, subresources ...string) (runtime.Object, error)
	Apply(namespace, resource, name string, obj *unstructured.Unstructured) (*unstructured.Unstructured, bool, error)
	Delete(namespace, resource, name string) error
	Watch(namespace string, resource, resourceVersion string, timeoutSeconds int64, selector interface{}) (<-chan watch.Event, error)
}

func NewIDataSource(cfg *configure.InstallConfigure) IDataSource {
	return &IDataSourceImpl{cfg}
}

type IDataSourceImpl struct {
	*configure.InstallConfigure
}

func (i *IDataSourceImpl) List(namespace, resource, flag string, pos, size int64, selector interface{}) (*unstructured.UnstructuredList, error) {
	var err error
	var items *unstructured.UnstructuredList
	opts := metav1.ListOptions{}

	if selector == nil || selector == "" {
		selector = labels.Everything()
	}
	switch selector.(type) {
	case labels.Selector:
		opts.LabelSelector = selector.(labels.Selector).String()
	case string:
		if selector != "" {
			opts.LabelSelector = selector.(string)
		}
	}

	if flag != "" {
		opts.Continue = flag
	}
	if size > 0 {
		opts.Limit = size + pos
	}
	gvr, err := i.GetGvr(resource)
	if err != nil {
		return nil, err
	}
	items, err = i.CacheInformerFactory.
		Interface.
		Resource(gvr).
		Namespace(namespace).
		List(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (i *IDataSourceImpl) Get(namespace, resource, name string, subresources ...string) (runtime.Object, error) {
	gvr, err := i.GetGvr(resource)
	if err != nil {
		return nil, err
	}
	object, err := i.CacheInformerFactory.
		Interface.
		Resource(gvr).
		Namespace(namespace).
		Get(context.Background(), name, metav1.GetOptions{}, subresources...)
	if err != nil {
		return nil, err
	}
	return object, nil
}

func (i *IDataSourceImpl) Apply(namespace, resource, name string, obj *unstructured.Unstructured) (*unstructured.Unstructured, bool, error) {
	panic("implement me")
}

func (i *IDataSourceImpl) Delete(namespace, resource, name string) error {
	gvr, err := i.GetGvr(resource)
	if err != nil {
		return  err
	}
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return i.
			CacheInformerFactory.
			Interface.
			Resource(gvr).
			Namespace(namespace).
			Delete(context.Background(), name, metav1.DeleteOptions{})
	})
	return retryErr
}

func (i *IDataSourceImpl) Watch(namespace string, resource, resourceVersion string, timeoutSeconds int64, selector interface{}) (<-chan watch.Event, error) {
	opts := metav1.ListOptions{}
	var err error

	if selector == nil || selector == "" {
		selector = labels.Everything()
	}
	switch selector.(type) {
	case labels.Selector:
		opts.LabelSelector = selector.(labels.Selector).String()
	case string:
		if selector != "" {
			opts.LabelSelector = selector.(string)
		}
	}

	if timeoutSeconds > 0 {
		opts.TimeoutSeconds = &timeoutSeconds
	}

	if resourceVersion != "" {
		opts.ResourceVersion = resourceVersion
	}
	gvr, err := i.GetGvr(resource)
	if err != nil {
		return nil, err
	}
	recv, err := i.CacheInformerFactory.
		Interface.
		Resource(gvr).
		Namespace(namespace).
		Watch(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	return recv.ResultChan(), nil
}
