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
	// CodeType language type
	CodeType string `json:"codeType"`
	// Type language type
	Type string `json:"type"`
	// RetryCount none
	RetryCount uint32 `json:"retryCount"`
	// Output output image repository
	Output string `json:"output"`
}

type RequestCd struct {
	FlowId     string   `json:"flowId"`
	StepName   string   `json:"stepName"`
	AckStates  []string `json:"ackStates"` //(SUCCESS | FAIL);
	UUID       string   `json:"uuid"`
	RetryCount uint32   `json:"retryCount"`

	ServiceName     string `json:"serviceName"`
	ServiceImage    string `json:"serviceImage"`
	DeployNamespace string `json:"deployNamespace"`

	ArtifactInfo map[string]interface{} `json:"artifactInfo"`

	DeployType  string `json:"DeployType"`
	CPULimit    string `json:"cpuLimit"`
	MEMLimit    string `json:"memLimit"`
	CPURequests string `json:"cpuRequests"`
	MEMRequests string `json:"memRequests"`
	Replicas    uint32 `json:"replicas"`
}

type Response struct {
	FlowId   string `json:"flowId"`
	StepName string `json:"stepName"`
	AckState string `json:"ackState"`
	UUID     string `json:"uuid"`
	Done     bool   `json:"done"`
}
