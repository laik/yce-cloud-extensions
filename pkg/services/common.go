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
)

var (
	BuildToolImage = "yametech/kaniko:v0.24.0"
	DestRepoUrl    = "harbor.ym/yce-cloud-extensions"
	CacheRepoUrl   = "yce-cloud-extensions-repo-cache"

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

type Parameter struct {
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

var _ io.Writer = &Output{}

type Output struct{ Data []byte }

func (o *Output) Write(p []byte) (n int, err error) {
	o.Data = append(o.Data, p...)
	if len(o.Data) < 1 {
		err = fmt.Errorf("can't not copy")
	}
	return
}

func Render(p *Parameter, tpl string) (*unstructured.Unstructured, error) {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return nil, err
	}
	o := &Output{}
	if err := t.Execute(o, p); err != nil {
		return nil, err
	}

	object := make(map[string]interface{})
	if err := yaml.Unmarshal(o.Data, &object); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: object}, nil
}
