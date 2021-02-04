package services

import (
	"flag"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
	"text/template"
)

const (
	TaskName               = "yce-cloud-extensions-task"
	PipelineName           = "yce-cloud-extensions-pipeline"
	PipelineGraphName      = "yce-cloud-extensions-graph"
	TektonGitConfigName    = "yce-cloud-extensions-git-config"
	TektonDockerConfigName = "yce-cloud-extensions-docker-config"

	// For java template
	JavaTaskName          = "yce-cloud-extensions-java-task"
	JavaPipelineGraphName = "yce-cloud-extensions-java-graph"
	JavaPipelineName      = "yce-cloud-extensions-java-pipeline"

	// For Unit template
	UnitTaskName          = "yce-cloud-extensions-unit-task"
	UnitPipelineGraphName = "yce-cloud-extensions-unit-graph"
	UnitPipelineName      = "yce-cloud-extensions-unit-pipeline"
)

var (
	BuildToolImage = "yametech/kaniko:v0.24.0"
	DestRepoUrl    = "harbor.ym/yce-cloud-extensions"
	CacheRepoUrl   = "harbor.ym/yce-cloud-extensions-repo-cache"

	// git server config
	ConfigGitUrl      = "http://git.ym"
	ConfigGitUser     = "yce-cloud-extensions" //"yce-cloud-extensions"
	ConfigGitPassword = "admin12345!QAZ"       //"admin12345!QAZ"

	ConfigRegistryUrl      = "http://harbor.ym"
	ConfigRegistryUserName = "yce-cloud-extensions"
	ConfigRegistryPassword = "admin12345!QAZ"
)

func init() {
	flag.StringVar(&ConfigGitUrl, "git-server", ConfigGitUrl, "-git-server http://git.ym")
	flag.StringVar(&ConfigGitUser, "git-user", ConfigGitUser, "-git-user username")
	flag.StringVar(&ConfigGitPassword, "git-password", ConfigGitPassword, "-git-password password")

	flag.StringVar(&ConfigRegistryUrl, "registry-server", ConfigRegistryUrl, "-registry-server http://harbor.ym")
	flag.StringVar(&ConfigRegistryUserName, "registry-user", ConfigRegistryUserName, "-registry-user username")
	flag.StringVar(&ConfigRegistryPassword, "registry-password", ConfigRegistryPassword, "-registry-password password")

	flag.StringVar(&BuildToolImage, "build-tool-image", BuildToolImage, "-build-tool-image yametech/kaniko:v0.24.0")
	flag.StringVar(&DestRepoUrl, "dest-repo", DestRepoUrl, "-dest-repo harbor.ym/yce-cloud-extensions")
	flag.StringVar(&CacheRepoUrl, "cache-repo", CacheRepoUrl, "-cache-repo harbor.ym/yce-cloud-extensions-repo-cache")
}

var _ io.Writer = &Output{}

type Output struct{ Data []byte }

func (o *Output) Write(p []byte) (n int, err error) {
	o.Data = append(o.Data, p...)
	if len(o.Data) < 1 {
		err = fmt.Errorf("can't not copy")
	}
	return
}

func Render(data interface{}, tpl string) (*unstructured.Unstructured, error) {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return nil, err
	}
	o := &Output{}
	if err := t.Execute(o, data); err != nil {
		return nil, err
	}

	object := make(map[string]interface{})
	if err := yaml.Unmarshal(o.Data, &object); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: object}, nil
}
