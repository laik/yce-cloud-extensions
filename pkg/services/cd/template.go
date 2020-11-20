package cd

import (
	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
	"gopkg.in/yaml.v2"
)

const stoneTpl = `kind: Stone
apiVersion: nuwa.nip.io/v1
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  labels:
    app: {{.Name}}
    app-uuid: {{.UUID}}
spec:
  template:
    metadata:
      name: {{.Name}}
      labels:
        app: {{.Name}}
        app-uuid: {{.UUID}}
    spec:
      containers:
        - name: {{.Name}}
          image: {{.Image}}
          resources:
            limits:
              cpu: {{.CpuLimit}}
              memory: {{.MemoryLimit}}
            requests:
              cpu: {{.CpuRequests}}
              memory: {{.MemoryRequests}}
          imagePullPolicy: Always
  strategy: Release
  {{.Coordinates}}
  service:
    ports:
{{range .ServicePorts}}
      - name: {{.Name}}
        protocol: {{.Protocol}}
        port: {{.Port}}
        targetPort: {{.TargetPort}}
{{end}}
    type: {{.ServiceType}}`

type params struct {
	Namespace      string
	Name           string
	Image          string
	CpuLimit       string
	MemoryLimit    string
	CpuRequests    string
	MemoryRequests string
	ServicePorts   []v1.ServicePorts
	ServiceType    string
	UUID           string
	Coordinates    string
}

type namespaceResourceLimit struct {
	Rack string `json:"rack"`
	Host string `json:"host"`
	Zone string `json:"zone"`
}

type namespaceResourceLimitSlice []namespaceResourceLimit

func (n *namespaceResourceLimitSlice) GroupBy() map[string][]namespaceResourceLimit {
	result := make(map[string][]namespaceResourceLimit)
	for _, v := range *n {
		if result[v.Zone] == nil {
			result[v.Zone] = make([]namespaceResourceLimit, 0)
		}
		result[v.Zone] = append(result[v.Zone], v)
	}
	return result
}

type resourceLimitStruct struct {
	Group    string                   `json:"group" yaml:"group"`
	ZoneSet  []namespaceResourceLimit `json:"zoneset" yaml:"zoneset"`
	Replicas uint32                   `json:"replicas" yaml:"replicas"`
}

type resourceLimitStructSlice struct {
	Coordinates []resourceLimitStruct `json:"coordinates" yaml:"coordinates"`
}

func createResourceLimitStructs(m map[string][]namespaceResourceLimit, replicas uint32) ([]byte, error) {
	coordinates := make([]resourceLimitStruct, 0)
	for k, v := range m {
		coordinates = append(
			coordinates,
			resourceLimitStruct{
				Group:    k,
				ZoneSet:  v,
				Replicas: replicas,
			})
	}
	return yaml.Marshal(&resourceLimitStructSlice{coordinates})
}
