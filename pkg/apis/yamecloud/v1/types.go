package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	SuccessState = "SUCCESS"
	FailState    = "FAIL"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
type CI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CISpec `json:"spec"`
}

type CISpec struct {
	GitURL     *string `json:"gitUrl"`
	Branch     *string `json:"branch"`
	CommitID   *string `json:"commitId"`
	CodeType   string  `json:"codeType"`
	RetryCount *uint32 `json:"retryCount"`
	Output     *string `json:"output"`
	Done       bool    `json:"done"`
	// fsm request field
	FlowId    *string  `json:"flowId"`
	StepName  *string  `json:"stepName"`
	AckStates []string `json:"ackStates"`
	UUID      *string  `json:"uuid"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CI `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
type CD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CDSpec `json:"spec"`
}

type CDSpec struct {
	ServiceName     *string       `json:"serviceName"`
	ServiceImage    *string       `json:"serviceImage"`
	DeployNamespace *string       `json:"deployNamespace"`
	ArtifactInfo    *ArtifactInfo `json:"artifactInfo"`
	DeployType      *string       `json:"deployType"`
	CPULimit        *string       `json:"cpuLimit"`
	MEMLimit        *string       `json:"memLimit"`
	CPURequests     *string       `json:"cpuRequests"`
	MEMRequests     *string       `json:"memRequests"`
	Replicas        uint32        `json:"replicas"`
	Done            bool          `json:"done"`
	FlowId          *string       `json:"flowId"`
	StepName        *string       `json:"stepName"`
	AckStates       []string      `json:"ackStates"`
	UUID            *string       `json:"uuid"`
}

type ArtifactInfo struct {
	Command      []string       `json:"command"`
	Arguments    []string       `json:"arguments"`
	Environments []Envs         `json:"environments"`
	ServicePorts []ServicePorts `json:"servicePorts"`
}

type ServicePorts struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
}

type Envs struct {
	Name  string `json:"name"`
	Kvalue string `json:"kvalue"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CD `json:"items"`
}
