package tools

import (
	"encoding/json"
	"fmt"
	gyaml "github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sort"
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

func ExtractProject(git string) (string, error) {
	if !strings.HasSuffix(git, ".git") {
		return "", fmt.Errorf("git addr illegal (%s)", git)
	}

	_slice := strings.Split(strings.TrimSuffix(git, ".git"), "/")
	if len(_slice) < 1 {
		return "", fmt.Errorf("git addr illegal (%s)", git)
	}
	return strings.ToLower(strings.Replace(_slice[len(_slice)-1], "_", "-", -1)), nil
}

// harbor.ym/devops/devops-taiyi-ui-k8s@sha256:fba94e0ce9ea241fa1047ea7f84b616093ff6a5d30d193bee2b3431f9e88d33c
func ExtractService(ServiceName string) (string, error) {
	if !strings.Contains(ServiceName, "sha256") {
		return "", fmt.Errorf("ServiceName addr illegal (%s)", ServiceName)
	}

	sliceUrl := strings.Split(ServiceName, "@sha256")
	if len(sliceUrl) < 1 {
		return "", fmt.Errorf("ServiceName addr illegal (%s)", ServiceName)
	}
	url := sliceUrl[0]
	_slice := strings.Split(url, "/")
	if len(_slice) < 1 {
		return "", fmt.Errorf("url addr illegal (%s)", url)
	}
	return _slice[len(_slice)-1], nil
}

func CompareSpecByUnstructured(source, target *unstructured.Unstructured) bool {
	if source == nil || target == nil {
		return false
	}
	srcUnstructuredSpec, exist := source.Object["spec"]
	if !exist {
		return false
	}
	targetUnstructuredSpec, exist := target.Object["spec"]
	if !exist {
		return false
	}
	if !reflect.DeepEqual(srcUnstructuredSpec, targetUnstructuredSpec) {
		return false
	}
	return true
}

func ContainStringItem(list []string, item string) bool {
	if sort.SearchStrings(list, item) >= 0 {
		return true
	}
	return false
}

func CloneNewObject(src *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	bytes, err := src.MarshalJSON()
	if err != nil {
		return nil, err
	}

	delete := func(res string, paths []string) (string, error) {
		var err error
		for _, path := range paths {
			res, err = sjson.Delete(res, path)
			if err != nil {
				return "", err
			}
		}
		return res, nil
	}

	dest, err := delete(string(bytes), []string{
		"metadata.creationTimestamp",
		"metadata.generation",
		"metadata.managedFields",
		"metadata.resourceVersion",
		"metadata.selfLink",
		"metadata.uid",
		"status",
	})

	obj := make(map[string]interface{})
	if err := json.Unmarshal([]byte(dest), &obj); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: obj}, nil
}

func SetObjectOwner(object []byte, apiVersion, kind, name, uid string) (*unstructured.Unstructured, error) {
	type ownerReference struct {
		ApiVersion         string `json:"apiVersion"`
		Kind               string `json:"kind"`
		Name               string `json:"name"`
		UID                string `json:"uid"`
		Controller         bool   `json:"controller"`
		BlockOwnerDeletion bool   `json:"blockOwnerDeletion"`
	}

	s, err := sjson.Set(string(object), "metadata.ownerReferences", []ownerReference{{
		ApiVersion:         apiVersion,
		Kind:               kind,
		Name:               name,
		UID:                uid,
		Controller:         false,
		BlockOwnerDeletion: false,
	}})
	if err != nil {
		return nil, err
	}

	obj := make(map[string]interface{})
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: obj}, nil
}

func SetYamlValue(yamlData []byte, path string, value interface{}) ([]byte, error) {
	jsonData, err := gyaml.YAMLToJSON(yamlData)
	if err != nil {
		return []byte(""), err
	}
	j, err := sjson.Set(string(jsonData), path, value)
	if err != nil {
		return []byte(""), err
	}
	return gyaml.JSONToYAML([]byte(j))
}

func GetYamlValue(yamlData []byte, path string) (gjson.Result, error) {
	jsonData, err := gyaml.YAMLToJSON(yamlData)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.Get(string(jsonData), path), nil
}
