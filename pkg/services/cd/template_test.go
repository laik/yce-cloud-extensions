package cd

import (
	"fmt"
	v1 "github.com/laik/yce-cloud-extensions/pkg/apis/yamecloud/v1"
	"github.com/laik/yce-cloud-extensions/pkg/services"
	"gopkg.in/yaml.v2"
	"reflect"
	"testing"
	"text/template"
)

var tt = template.New("template")

func TestStoneConstructor(t *testing.T) {
	o := &services.Output{}
	tt = template.Must(tt.Parse(stoneTpl))
	err := tt.Execute(o,
		&params{
			Namespace:      "ns",
			Name:           "test1",
			Image:          "abc",
			CpuLimit:       "100m",
			MemoryLimit:    "30m",
			CpuRequests:    "1000m",
			MemoryRequests: "300m",
			ConfigVolumes: []v1.ConfigVolumes{
				{MountName: "volume-test",
					MountPath: "/var/www/",
					Kind:      "configmap",
					CMItems:   []v1.CMItems{}},
				{
					MountName: "data1",
					SubPath:   "100Mi",
					Kind:      "storage",
					MountPath: "/data",
				},
				{
					MountName: "data2",
					SubPath:   "200Mi",
					Kind:      "storage",
					MountPath: "/dxp",
				},
			},
			StorageClass: "kube-ceph-xfs",
			ServicePorts: []v1.ServicePorts{
				{Name: "port", Protocol: "TCP", Port: 80, TargetPort: 80},
			},
			ServiceType: "ClusterIP",
			UUID:        "abc123",
			Coordinates: []ResourceLimitStruct{
				{Group: "B", Replicas: 1, ZoneSet: []NamespaceResourceLimit{
					{Rack: "W-01", Host: "node3", Zone: "B"},
					{Rack: "S-05", Host: "node2", Zone: "B"},
					{Rack: "S-02", Host: "node4", Zone: "B"},
				}},
			},
			Commands: []string{"start"},
			Args:     []string{"abc"},
			CDName:   "abc",
		})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s", o.Data)

	expected := `kind: Stone
apiVersion: nuwa.nip.io/v1
metadata:
  name: test1
  namespace: ns
  labels:
    app: test1
    app-uuid: abc123
    yce-cloud-extensions: abc
spec:
  template:
    metadata:
      name: test1
      labels:
        app: test1
        app-uuid: abc123
    spec:
      containers:
        - name: test1
          image: abc
          command:
            - 'start'
          args:
            - 'abc'
          resources:
            limits:
              cpu: 100m
              memory: 30m
            requests:
              cpu: 1000m
              memory: 300m
          imagePullPolicy: Always
  strategy: Release
  coordinates:
    - group: B
      zoneset:
        - zone: B
          rack: W-01
          host: node3
        - zone: B
          rack: S-05
          host: node2
        - zone: B
          rack: S-02
          host: node4
      replicas: 1
  service:
    ports:
      - name: port
        protocol: TCP
        port: 80
        targetPort: 80
    type: ClusterIP`

	src, dest := make(map[string]interface{}), make(map[string]interface{})
	if err, err1 := yaml.Unmarshal([]byte(expected), src), yaml.Unmarshal(o.Data, dest); err != nil || err1 != nil {
		if err != nil {
			t.Fatal(err)
		}
		if err1 != nil {
			t.Fatal(err1)
		}
	}

	if !reflect.DeepEqual(src, dest) {
		t.Fatal("expect not equal")
	}

}

func TestConfigMapConstructor(t *testing.T) {
	o := &services.Output{}
	tt = template.Must(tt.Parse(configMapTpl))
	err := tt.Execute(o,
		&params{
			Name: "dxp",
			ConfigVolumes: []v1.ConfigVolumes{
				{MountName: "volume-test",
					MountPath: "/var/www/",
					CMItems: []v1.CMItems{
						{VolumePath: "main.html",
							VolumeName: "html",
							VolumeData: "hello world"},
						{VolumePath: "ok.py",
							VolumeName: "py",
							VolumeData: "go"},
					}},
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s", o.Data)

	expected := `kind: ConfigMap
apiVersion: nuwa.nip.io/v1
metadata:
  name: dxp
data:
  html: hello world
  py: go
  `
	src, dest := make(map[string]interface{}), make(map[string]interface{})
	if err, err1 := yaml.Unmarshal([]byte(expected), src), yaml.Unmarshal(o.Data, dest); err != nil || err1 != nil {
		if err != nil {
			t.Fatal(err)
		}
		if err1 != nil {
			t.Fatal(err1)
		}
	}

	if !reflect.DeepEqual(src, dest) {
		t.Fatal("expect not equal")
	}

}
