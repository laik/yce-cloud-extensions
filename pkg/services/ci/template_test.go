package ci

import (
	"strings"
	"testing"
	"text/template"
)


var tt = template.New("template")

func TestTaskConstructor(t *testing.T) {
	o := &output{}
	tt = template.Must(tt.Parse(taskTpl))
	if err := tt.Execute(o, &parameter{Namespace: "test"}); err != nil {
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
        - name: DOCKER_CONFIG
          value: /tekton/home/.docker
      image: $(params.build_tool_image)
      name: main
      resources: {}
      command: []
      script: ''
      workingDir: ''
  volumes:
    - emptyDir: {}
      name: build-path`
	data := string(o.data)
	if !strings.EqualFold(data, expected) {
		t.Fatal("expect not equal")
	}
}

func TestPipelineConstructor(t *testing.T) {
	o := &output{}
	tt = template.Must(tt.Parse(pipelineTpl))
	if err := tt.Execute(o, &parameter{
		Namespace:      "test",
		ProjectName:    "test_project_name",
		ProjectVersion: "v0.0.1",
		BuildToolImage: "yametech/kaniko:v0.24.0",
		DestRepoUrl:    "http://harbor.ym/test",
		CacheRepoUrl:   "cache.compass.ym:5000",
	}); err != nil {
		t.Fatal(err)
	}
	expected := `apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  annotations:
    fuxi.nip.io/tektongraphs: yce-cloud-extensions-pipeline-default
    namespace: test
  labels:
    namespace: test
  name: yce-cloud-extensions-pipeline
  namespace: test-ops
spec:
  resources:
    - name: git-addr
      type: git
  tasks:
    - name: yce-cloud-extensions-task
      params:
        - name: project_name
          value: test_project_name
        - name: project_version
          value: v0.0.1
        - name: build_tool_image
          value: yametech/kaniko:v0.24.0
        - name: dest_repo_url
          value: http://harbor.ym/test
        - name: cache_repo_url
          value: cache.compass.ym:5000
      resources:
        inputs:
          - name: git
            resource: git-addr
      taskRef:
        kind: Task
        name: yce-cloud-extensions-task`

	if !strings.EqualFold(string(o.data), expected) {
		t.Fatal("expect not equal")
	}
}
