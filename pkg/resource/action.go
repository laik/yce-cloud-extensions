package resource

type Request struct {
	FlowId    string   `json:"flowId"`
	StepName  string   `json:"stepName"`
	AckStates []string `json:"ackStates"` //(SUCCESS | FAIL);
	UUID      string   `json:"uuid"`
	// GitUrl git url address
	GitUrl string `json:"gitUrl"`
	// Branch git Branch
	Branch string `json:"branch"`
	// CommitID the git commit id
	CommitID string `json:"commitId"`
	// Type language type
	Type string `json:"type"`
	// RetryCount none
	RetryCount uint32 `json:"retryCount"`
	// Output output image repository
	Output string `json:"output"`
	///////

}

type RequestCd struct {
	FlowId     string   `json:"flowId"`
	StepName   string   `json:"stepName"`
	AckStates  []string `json:"ackStates"` //(SUCCESS | FAIL);
	UUID       string   `json:"uuid"`
	RetryCount uint32   `json:"retryCount"`

	ServiceName     string            `json:"serviceName"`
	DeployNamespace string            `json:"deployNamespace"`
	ArtifactInfo    map[string]string `json:"artifactInfo"`
	DeployType      string            `json:"DeployType"`
}

type Response struct {
	FlowId   string `json:"flowId"`
	StepName string `json:"stepName"`
	AckState string `json:"ackState"`
	UUID     string `json:"uuid"`
	Done     bool   `json:"done"`
}
