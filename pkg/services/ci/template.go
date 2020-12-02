package ci

const (
	graphTpl = `kind: TektonGraph
apiVersion: fuxi.nip.io/v1
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  labels:
    namespace: {{.Namespace}}
spec:
  data: >-
    {"nodes":[{"id":"1-1","x":20,"y":20,"role":0,"taskName":"yce-cloud-extensions-task","anchorPoints":[[0,0.5],[1,0.5]],"addnode":true,"subnode":true,"type":"pipeline-node","linkPoints":{"right":true,"left":true},"style":{}}],"edges":[],"combos":[],"groups":[]}
  width: 1629
  height: 592`

	pipelineTpl = `apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  annotations:
    fuxi.nip.io/tektongraphs: {{.PipelineGraph}}
    namespace: {{.Namespace}}
  labels:
    namespace: {{.Namespace}}
  name: {{.Name}}
  namespace: {{.Namespace}}
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
    - default: ''
      name: code_type
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
        - name: code_type
          value: $(params.code_type)
      resources:
        inputs:
          - name: git
            resource: git-addr
      taskRef:
        kind: Task
        name: {{.TaskName}}`

	taskTpl = `apiVersion: tekton.dev/v1alpha1
kind: Task
metadata:
  labels:
    namespace: {{.Namespace}}
  name: yce-cloud-extensions-task
  namespace: {{.Namespace}}
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
    - default: none
      name: code_type
      type: string
  resources:
    inputs:
      - name: git
        type: git
    outputs: []
  steps:
    - args:
        - '-url'
        - /workspace/git
        - '-codetype'
        - $(params.code_type)
      env:
        - name: DOCKER_CONFIG
          value: /tekton/home/.docker
      image: 'yametech/checkdocker:v0.1.0'
      name: step1
      resources: {}
    - args:
        - '--dockerfile=/workspace/git/Dockerfile'
        - '--context=/workspace/git'
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
      name: step2
      resources: {}
      command: []
      script: ''
      workingDir: ''
  volumes:
    - emptyDir: {}
      name: build-path`

	pipelineResourceTpl = `kind: PipelineResource
apiVersion: tekton.dev/v1alpha1
metadata:
  labels:
    namespace: {{.Namespace}}
  name: {{.Name}}
  namespace: {{.Namespace}}
spec:
  params:
    - name: url
      value: {{.GitUrl}}
    - name: revision
      value: {{.Branch}}
  type: git`

	pipelineRunTpl = `apiVersion: tekton.dev/v1alpha1
kind: PipelineRun
metadata:
  annotations:
    fuxi.nip.io/run-tektongraphs: {{.PipelineRunGraph}}
    fuxi.nip.io/tektongraphs: {{.PipelineGraph}}
    namespace: {{.Namespace}}
  labels:
    namespace: {{.Namespace}}
    tekton.dev/pipeline: {{.PipelineName}}
  name: {{.Name}}
  namespace: {{.Namespace}}
spec:
  params:
    - name: project_name
      value: {{.ProjectName}}
    - name: project_version
      value: {{.ProjectVersion}}
    - name: build_tool_image
      value: {{.BuildToolImage}}
    - name: code_type
      value: {{.CodeType}}
    - name: dest_repo_url
      value: {{.DestRepoUrl}}
    - name: cache_repo_url
      value: {{.CacheRepoUrl}}
  pipelineRef:
    name: {{.PipelineName}}
  resources:
    - name: git-addr
      resourceRef:
        name: {{.PipelineResourceName}}
  serviceAccountName: default
  timeout: 1h0m0s`

	configGitTpl = `apiVersion: v1
data:
  password: {{.GitPassword}}
  username: {{.GitUsername}}
kind: Secret
metadata:
  annotations:
    tekton.dev/git-0: {{.ConfigGitUrl}}
  labels:
    mount: "1"
    tekton: "1"
  name: {{.Name}}
  namespace: {{.Namespace}}
type: kubernetes.io/basic-auth`

	configRegistryTpl = `apiVersion: v1
kind: Secret
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  labels:
    mount: '1'
    tekton: '1'
  annotations:
    tekton.dev/docker-0: {{.RegistryRepoUrl}}
data:
  password: {{.RegistryPassword}}
  username: {{.RegistryUsername}}
type: kubernetes.io/basic-auth`
)

type params struct {
	// common
	Namespace string
	Name      string
	// pipelineResourceTpl && pipelineTpl
	GitUrl string
	Branch string
	//Retries uint64
	// graphTpl
	ApiVersion                string
	PipelineOrPipelineRunName string
	Uid                       string
	// pipelineRunTpl
	PipelineRunGraph     string
	PipelineGraph        string
	PipelineResourceName string
	PipelineName         string
	ProjectName          string
	CodeType             string
	ProjectVersion       string
	BuildToolImage       string
	DestRepoUrl          string
	CacheRepoUrl         string
	TaskName             string
	// configGitTpl
	ConfigGitUrl string
	GitUsername  string
	GitPassword  string
	// configRegistryTpl
	RegistryRepoUrl  string
	RegistryPassword string
	RegistryUsername string
}
