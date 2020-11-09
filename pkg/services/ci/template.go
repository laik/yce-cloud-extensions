package ci

const (
	graphTpl = `kind: TektonGraph
apiVersion: fuxi.nip.io/v1
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  labels:
    namespace: yce
  ownerReferences:
    - apiVersion: {{.ApiVersion}}
      kind: Pipeline
      name: {{.PipelineOrPipelineRunName}}
      uid: {{.Uid}}
      controller: false
      blockOwnerDeletion: false
spec:
  data: >-
    {"nodes":[{"id":"1-1","x":20,"y":20,"role":0,"taskName":"yce-cloud-extensions-task","anchorPoints":[[0,0.5],[1,0.5]],"addnode":true,"subnode":true,"type":"pipeline-node","linkPoints":{"right":true,"left":true},"style":{}}],"edges":[],"combos":[],"groups":[]}
  width: 1629
  height: 592`

	pipelineTpl = `apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  annotations:
    fuxi.nip.io/tektongraphs: yce-cloud-extensions-pipeline-default
    namespace: {{.Namespace}}
  labels:
    namespace: {{.Namespace}}
  name: {{.Name}}
  namespace: {{.Namespace}}
spec:
  resources:
    - name: git-addr
      type: git
  tasks:
    - name: yce-cloud-extensions-task
      params:
        - name: project_name
          value: {{.ProjectName}}
        - name: project_version
          value: {{.ProjectVersion}}
        - name: build_tool_image
          value: {{.BuildToolImage}}
        - name: dest_repo_url
          value: {{.DestRepoUrl}}
        - name: cache_repo_url
          value: {{.CacheRepoUrl}}
      resources:
        inputs:
          - name: git
            resource: git-addr
      taskRef:
        kind: Task
        name: yce-cloud-extensions-task`

	taskTpl = `apiVersion: tekton.dev/v1alpha1
kind: Task
metadata:
  labels:
    namespace: {{.Namespace}}
  name: yce-cloud-extensions-task
  namespace: {{.Namespace}}-ops
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

	pipelineResourceTpl = `kind: PipelineResource
apiVersion: tekton.dev/v1alpha1
metadata:
  labels:
    namespace: {{.Namespace}}
  name: {{.Name}}}
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
    namespace: {{.Namepsace}}
  labels:
    namespace: {{.Namepsace}}
    tekton.dev/pipeline: {{.PipelineName}}
  name: {{.PipelineRunName}}
  namespace: {{.Namepsace}}-ops
spec:
  pipelineRef:
    name: {{.PipelineName}}
  resources:
    - name: git-addr
      resourceRef:
        name: {{.PipelineResourceName}}
  serviceAccountName: default
  timeout: 1h0m0s`

	configTpl = `apiVersion: v1
data:
  password: {{.Password}}
  username: {{.Username}}
kind: Secret
metadata:
  annotations:
    tekton.dev/git-0: {{.ConfigGitUrl}}
  labels:
    mount: "1"
    tekton: "1"
  name: {{.ConfigGitName}}
  namespace: {{.Namespace}}
type: kubernetes.io/basic-auth`
)
