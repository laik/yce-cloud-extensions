package ci

import (
	"github.com/laik/yce-cloud-extensions/pkg/services"
	"strings"
	"testing"
	"text/template"
)

var tt = template.New("template")

func TestTaskConstructor(t *testing.T) {
	o := &services.Output{}
	tt = template.Must(tt.Parse(taskTpl))
	if err := tt.Execute(o, &services.Parameter{Namespace: "test"}); err != nil {
		t.Fatal(err)
	}
	expected := `apiVersion: tekton.dev/v1alpha1
kind: Task
metadata:
  labels:
    namespace: test
  name: yce-cloud-extensions-task
  namespace: test-ops
spec:
  params:
    - default: none
      name: project_name
      type: string
    - default: none
      name: project_version
      type: string
    - default: 'yametech/kaniko:v0.24.0'
      name: build_tool_image
      type: string
    - default: none
      name: dest_repo_url
      type: string
    - default: none
      name: cache_repo_url
      type: string
  resources:
    inputs:
      - name: git
        type: git
    outputs: []
  steps:
    - args:
        - '--dockerfile=/workspace/$(params.project_name)/Dockerfile'
        - '--context=/workspace/$(params.project_name)'
        - '--insecure'
        - '--force'
        - '--destination=$(params.dest_repo_url)/$(params.project_name):$(params.project_version)'
        - '--cache=true'
        - '--skip-tls-verify'
        - '--cache-repo=$(params.cache_repo_url)/$(params.project_name)-cache'
        - '--skip-unused-stages=true'
      env:
        - name: "DOCKER_CONFIG"
          value: "/tekton/home/.docker"
      image: $(params.build_tool_image)
      name: main
      resources: {}
      command: []
      script: ''
      workingDir: ''
  volumes:
    - emptyDir: {}
      name: build-path`
	data := string(o.Data)
	if !strings.EqualFold(data, expected) {
		t.Fatal("expect not equal")
	}

}

func TestPipelineConstructor(t *testing.T) {
	o := &services.Output{}
	tt = template.Must(tt.Parse(pipelineTpl))
	if err := tt.Execute(o,
		&services.Parameter{
			Namespace:     "test-ops",
			Name:          "yce-cloud-extensions-pipeline",
			PipelineGraph: "my-graph",
			TaskName:      "my-task",
		}); err != nil {
		t.Fatal(err)
	}
	expected := `apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  annotations:
    fuxi.nip.io/tektongraphs: my-graph
    namespace: test-ops
  labels:
    namespace: test-ops
  name: yce-cloud-extensions-pipeline
  namespace: test-ops
spec:
  params:
    - default: ''
      name: project_name
      type: string
    - default: ''
      name: project_version
      type: string
    - default: ''
      name: build_tool_image
      type: string
    - default: ''
      name: dest_repo_url
      type: string
    - default: ''
      name: cache_repo_url
      type: string
  resources:
    - name: git-addr
      type: git
  tasks:
    - name: yce-cloud-extensions-task
      params:
        - name: project_name
          value: $(params.project_name)
        - name: project_version
          value: $(params.project_version)
        - name: build_tool_image
          value: $(params.build_tool_image)
        - name: dest_repo_url
          value: $(params.dest_repo_url)
        - name: cache_repo_url
          value: $(params.cache_repo_url)
      resources:
        inputs:
          - name: git
            resource: git-addr
      taskRef:
        kind: Task
        name: my-task`

	data := string(o.Data)
	if !strings.EqualFold(data, expected) {
		t.Fatal("expect not equal")
	}
}
