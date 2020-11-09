package ci

import (
	"flag"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"text/template"
)

const (
	taskName                  = "yce-cloud-extensions-task"
	pipelineName              = "yce-cloud-extensions-pipeline"
	pipelineGraphName         = "yce-cloud-extensions-graph"
	pipelineResourceNameModel = "yce-cloud-extensions-%s"
)

var (
	configGitUser     = "yce-cloud-extensions"
	configGitPassword = "admin12345!QAZ"
	buildToolImage    = "yametech/kaniko:v0.24.0"
	destRepoUrl       = "harbor.ym/yce-cloud-extensions"
	cacheRepoUrl      = "yce-cloud-extensions-repo-cache"
)

func init() {
	flag.StringVar(&configGitUser, "git-user", configGitUser, "-git-user username")
	flag.StringVar(&configGitPassword, "git-password", configGitPassword, "-git-password password")
	flag.StringVar(&buildToolImage, "build-tool-image", buildToolImage, "-build-tool-image yametech/kaniko:v0.24.0")
	flag.StringVar(&destRepoUrl, "dest-repo", destRepoUrl, "-dest-repo harbor.ym/yce-cloud-extensions")
	flag.StringVar(&cacheRepoUrl, "cache-repo", cacheRepoUrl, "-cache-repo harbor.ym/yce-cloud-extensions-repo-cache")
}

type parameter struct {
	// common
	Namespace string
	Name      string

	// pipelineResourceTpl
	GitUrl         string
	Branch         string
	ProjectName    string
	ProjectVersion string
	BuildToolImage string
	DestRepoUrl    string
	CacheRepoUrl   string

	// graphTpl
	ApiVersion                string
	PipelineOrPipelineRunName string
	Uid                       string

	// pipelineRunTpl
	PipelineRunGraph     string
	PipelineGraph        string
	PipelineResourceName string
	PipelineName         string
	PipelineRunName      string

	// configTpl
	ConfigGitUrl  string
	Username      string
	Password      string
	ConfigGitName string
}

var _ io.Writer = &output{}

type output struct{ data []byte }

func (o *output) Write(p []byte) (n int, err error) {
	o.data = append(o.data, p...)
	if len(o.data) < 1 {
		err = fmt.Errorf("can't not copy")
	}
	return
}

func render(p *parameter, tpl string) (*unstructured.Unstructured, error) {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return nil, err
	}
	o := &output{}
	if err := t.Execute(o, p); err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	if err := obj.UnmarshalJSON(o.data); err != nil {
		return nil, err
	}

	return obj, nil
}
