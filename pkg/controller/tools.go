package controller

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

func RuntimeObjectToInstance(object runtime.Object, target interface{}) error {
	bytesData, err := json.Marshal(object)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytesData, target)
}

func InstanceToUnstructured(object runtime.Object) (*unstructured.Unstructured, error) {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: unstructuredObj,
	}, nil
}

func UnstructuredListObjectToInstanceObjectList(obj *unstructured.UnstructuredList, targetObj interface{}) error {
	bytesData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytesData, targetObj)
}

func UnstructuredObjectToInstanceObj(obj *unstructured.Unstructured, targetObj interface{}) error {
	bytesData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytesData, targetObj)
}

func extractProject(git string) (string, error) {
	if !strings.HasSuffix(git, ".git") {
		return "", fmt.Errorf("git addr illegal (%s)", git)
	}

	_slice := strings.Split(strings.TrimSuffix(git, ".git"), "/")
	if len(_slice) < 1 {
		return "", fmt.Errorf("git addr illegal (%s)", git)
	}
	return _slice[len(_slice)-1], nil
}
