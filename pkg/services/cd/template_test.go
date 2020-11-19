package cd

import (
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
	if err := tt.Execute(o, &params{
		Namespace:      "ns",
		Name:           "test1",
		Image:          "abc",
		CpuLimit:       "100m",
		MemoryLimit:    "30m",
		CpuRequests:    "1000m",
		MemoryRequests: "300m",
		ServicePorts: []v1.ServicePorts{
			{Name: "port", Protocol: "TCP", Port: 80, TargetPort: 80},
		},
		ServiceType: "ClusterIP",
		UUID:        "abc123",
		Coordinates: `  coordinates:
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
      replicas: 1`,
	}); err != nil {
		t.Fatal(err)
	}
	expected := `kind: Stone
apiVersion: nuwa.nip.io/v1
metadata:
  name: test1
  namespace: ns
  labels:
    app: test1
    app-uuid: abc123
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
	if err, err1 := yaml.Unmarshal([]byte(expected), src), yaml.Unmarshal([]byte(expected), dest); err != nil || err1 != nil {
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
