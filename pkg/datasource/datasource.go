package datasource

import (
	"context"
	"github.com/laik/yce-cloud-extensions/pkg/configure"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
	"reflect"
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

func (i *IDataSourceImpl) Apply(namespace, resource, name string, obj *unstructured.Unstructured) (result *unstructured.Unstructured, isUpdate bool, err error) {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		gvr, err := i.GetGvr(resource)
		if err != nil {
			return err
		}
		ctx := context.Background()
		getObj, getErr := i.CacheInformerFactory.
			Interface.
			Resource(gvr).
			Namespace(namespace).
			Get(ctx, name, metav1.GetOptions{})

		if errors.IsNotFound(getErr) {
			newObj, createErr := i.CacheInformerFactory.
				Interface.
				Resource(gvr).
				Namespace(namespace).
				Create(ctx, obj, metav1.CreateOptions{})
			result = newObj
			return createErr
		}

		if getErr != nil {
			return getErr
		}

		compareObject(getObj, obj)

		newObj, updateErr := i.CacheInformerFactory.
			Interface.
			Resource(gvr).
			Namespace(namespace).
			Update(ctx, getObj, metav1.UpdateOptions{})

		result = newObj
		isUpdate = true
		return updateErr
	})
	err = retryErr

	return
}

func (i *IDataSourceImpl) Delete(namespace, resource, name string) error {
	gvr, err := i.GetGvr(resource)
	if err != nil {
		return err
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

func compareObject(getObj, obj *unstructured.Unstructured) {
	if !reflect.DeepEqual(getObj.Object["metadata"], obj.Object["metadata"]) {
		getObj.Object["metadata"] = compareMetadataLabelsOrAnnotation(
			getObj.Object["metadata"].(map[string]interface{}),
			obj.Object["metadata"].(map[string]interface{}),
		)
	}

	if !reflect.DeepEqual(getObj.Object["spec"], obj.Object["spec"]) {
		getObj.Object["spec"] = obj.Object["spec"]
	}

	// configMap
	if !reflect.DeepEqual(getObj.Object["data"], obj.Object["data"]) {
		getObj.Object["data"] = obj.Object["data"]
	}

	if !reflect.DeepEqual(getObj.Object["binaryData"], obj.Object["binaryData"]) {
		getObj.Object["binaryData"] = obj.Object["binaryData"]
	}

	if !reflect.DeepEqual(getObj.Object["stringData"], obj.Object["stringData"]) {
		getObj.Object["stringData"] = obj.Object["stringData"]
	}

	if !reflect.DeepEqual(getObj.Object["type"], obj.Object["type"]) {
		getObj.Object["type"] = obj.Object["type"]
	}

	if !reflect.DeepEqual(getObj.Object["secrets"], obj.Object["secrets"]) {
		getObj.Object["secrets"] = obj.Object["secrets"]
	}

	if !reflect.DeepEqual(getObj.Object["imagePullSecrets"], obj.Object["imagePullSecrets"]) {
		getObj.Object["imagePullSecrets"] = obj.Object["imagePullSecrets"]
	}
	// storageClass field
	if !reflect.DeepEqual(getObj.Object["provisioner"], obj.Object["provisioner"]) {
		getObj.Object["provisioner"] = obj.Object["provisioner"]
	}

	if !reflect.DeepEqual(getObj.Object["parameters"], obj.Object["parameters"]) {
		getObj.Object["parameters"] = obj.Object["parameters"]
	}

	if !reflect.DeepEqual(getObj.Object["reclaimPolicy"], obj.Object["reclaimPolicy"]) {
		getObj.Object["reclaimPolicy"] = obj.Object["reclaimPolicy"]
	}

	if !reflect.DeepEqual(getObj.Object["volumeBindingMode"], obj.Object["volumeBindingMode"]) {
		getObj.Object["volumeBindingMode"] = obj.Object["volumeBindingMode"]
	}
}

func compareMetadataLabelsOrAnnotation(old, new map[string]interface{}) map[string]interface{} {
	newLabels, exist := new["labels"]
	if exist {
		old["labels"] = newLabels
	}
	newAnnotations, exist := new["annotations"]
	if exist {
		old["annotations"] = newAnnotations
	}

	newOwnerReferences, exist := new["ownerReferences"]
	if exist {
		old["ownerReferences"] = newOwnerReferences
	}
	return old
}
