package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-nexus-operator/api/common"
)

// NexusScriptSpec defines the desired state of NexusScript.
type NexusScriptSpec struct {
	// Name is the id of the script.
	// Name should be unique across all scripts.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=512
	// +required
	// +kubebuilder:example="anonymous"
	Name string `json:"name"`

	// Content is the content of the script.
	// +required
	// +kubebuilder:example="security.setAnonymousAccess(Boolean.valueOf(args))"
	Content string `json:"content"`

	// Payload is the payload of the script.
	// +optional
	// +kubebuilder:example="true"
	Payload string `json:"payload,omitempty"`

	// Execute defines if script should be executed after creation.
	// +optional
	// +kubebuilder:default=false
	Execute bool `json:"execute"`

	// NexusRef is a reference to Nexus custom resource.
	// +required
	NexusRef common.NexusRef `json:"nexusRef"`
}

// NexusScriptStatus defines the observed state of NexusScript.
type NexusScriptStatus struct {
	// Value is a status of the script.
	// +optional
	Value string `json:"value,omitempty"`

	// Error is an error message if something went wrong.
	// +optional
	Error string `json:"error,omitempty"`

	// Executed defines if script was executed.
	Executed bool `json:"executed,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NexusScript is the Schema for the nexusscripts API.
type NexusScript struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusScriptSpec   `json:"spec,omitempty"`
	Status NexusScriptStatus `json:"status,omitempty"`
}

func (in *NexusScript) GetNexusRef() common.NexusRef {
	return in.Spec.NexusRef
}

// +kubebuilder:object:root=true

// NexusScriptList contains a list of NexusScript.
type NexusScriptList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NexusScript `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NexusScript{}, &NexusScriptList{})
}
