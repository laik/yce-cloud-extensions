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

// harbor.ym/devops/devops-taiyi-ui-k8s@sha256:fba94e0ce9ea241fa1047ea7f84b616093ff6a5d30d193bee2b3431f9e88d33c
func extractService(ServiceName string) (string, error) {
	if !strings.Contains(ServiceName, "sha256") {
		return "", fmt.Errorf("ServiceName addr illegal (%s)", ServiceName)
	}

	_slice_url := strings.Split(ServiceName, "@sha256")
	if len(_slice_url) < 1 {
		return "", fmt.Errorf("ServiceName addr illegal (%s)", ServiceName)
	}
	url := _slice_url[0]
	_slice := strings.Split(url, "/")
	if len(_slice) < 1 {
		return "", fmt.Errorf("url addr illegal (%s)", url)
	}
	return _slice[len(_slice)-1], nil
}
