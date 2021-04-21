package cd

import (
	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
)

const (
	stoneTpl = `kind: Stone
apiVersion: nuwa.nip.io/v1
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  labels:
    app: {{.Name}}
    app-uuid: {{.UUID}}
    yce-cloud-extensions: {{.CDName}}
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
          {{- if .Commands}}
          command:
          {{ range .Commands}}
            - {{.}}
          {{ end }}
          {{- end }}
          {{- if .Args}}
          args:
          {{ range .Args}}
            - {{.}}
          {{ end }}
          {{- end }}
          {{- if .Environments}}
          env:
          {{range .Environments}}
            - name: {{.Name}}
              value: {{.Envvalue}}
          {{ end }}
          {{- end }}
          resources:
            limits:
              cpu: {{.CpuLimit}}
              memory: {{.MemoryLimit}}
            requests:
              cpu: {{.CpuRequests}}
              memory: {{.MemoryRequests}}
          imagePullPolicy: {{.Policy}}
          {{- if .ConfigVolumes}}
          volumeMounts:
            {{range .ConfigVolumes}}
            - name: {{.MountName}}
              mountPath: {{.MountPath}}
              {{- if eq .Kind "configmap"}}
              subPath: {{.SubPath}}
              {{- end }}
            {{ end }}
          {{- end }}
      {{- if .ConfigVolumes}}
      volumes:
        {{range .ConfigVolumes}}
        {{- if eq .Kind "configmap"}}
        - name: {{.MountName}}
          configMap:
            name: {{$.Name}}
            {{- if .CMItems}}
            items:
              {{range .CMItems}}
              - key: {{.VolumeName}}
                path: {{.VolumePath}}
              {{ end }}
            {{- end }}
        {{- end }}
        {{ end }}
      {{- end }}
  {{- if eq .NeedStorage "true"}}
  volumeClaimTemplates:
  {{- end }}
  {{range .ConfigVolumes}}
  {{- if eq .Kind "storage"}}
    - metadata:
        name: {{.MountName}}
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: {{.SubPath}}
        storageClassName: {{$.StorageClass}}
  {{- end }}
  {{ end }}
  strategy: Release
  coordinates:
{{range .Coordinates}}
    - group: {{.Group}}
      zoneset:
{{range .ZoneSet}}
        - zone: {{.Zone}}
          rack: {{.Rack}}
          host: {{.Host}}
{{end}}
      replicas: {{.Replicas}}
{{end}}
  service:
    ports:
{{range .ServicePorts}}
      - name: {{.Name}}
        protocol: {{.Protocol}}
        port: {{.Port}}
        targetPort: {{.TargetPort}}
{{end}}
    type: {{.ServiceType}}`

	configMapTpl = `kind: ConfigMap
apiVersion: v1
metadata:
  name: {{.Name}}
data:
  {{range .ConfigVolumes}}
  {{range .CMItems}}
  {{.VolumeName}}: {{.VolumeData}}
  {{ end }}
  {{ end }}
`
)

type params struct {
	Namespace      string
	Name           string
	Image          string
	CpuLimit       string
	MemoryLimit    string
	CpuRequests    string
	MemoryRequests string
	Policy         string
	StorageClass   string
	Commands       []string
	Args           []string
	ServicePorts   []v1.ServicePorts
	ServiceType    string
	UUID           string
	Coordinates    []ResourceLimitStruct
	CDName         string
	Environments   []v1.Envs
	ConfigVolumes  []v1.ConfigVolumes
	NeedStorage    string
}

type NamespaceResourceLimit struct {
	Rack string `json:"rack"`
	Host string `json:"host"`
	Zone string `json:"zone"`
}

type NamespaceResourceLimitSlice []NamespaceResourceLimit

func (n *NamespaceResourceLimitSlice) GroupBy() map[string][]NamespaceResourceLimit {
	result := make(map[string][]NamespaceResourceLimit)
	for _, v := range *n {
		if result[v.Zone] == nil {
			result[v.Zone] = make([]NamespaceResourceLimit, 0)
		}
		result[v.Zone] = append(result[v.Zone], v)
	}
	return result
}

type ResourceLimitStruct struct {
	Group    string                   `json:"group" yaml:"group"`
	ZoneSet  []NamespaceResourceLimit `json:"zoneset" yaml:"zoneset"`
	Replicas uint32                   `json:"replicas" yaml:"replicas"`
}

func createResourceLimitStructs(m map[string][]NamespaceResourceLimit, replicas uint32) []ResourceLimitStruct {
	coordinates := make([]ResourceLimitStruct, 0)
	for k, v := range m {
		coordinates = append(
			coordinates,
			ResourceLimitStruct{
				Group:    k,
				ZoneSet:  v,
				Replicas: replicas,
			})
	}
	return coordinates
}
