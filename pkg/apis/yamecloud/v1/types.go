package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	Spec CISpec `json:"spec"`
}

type CDSpec struct {
	ServiceName     string            `json:"service_name"`
	DeployNamespace string            `json:"deploy_namespace"`
	ArtifactInfo    map[string]string `json:"artifact_info"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CD `json:"items"`
}
