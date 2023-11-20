package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NexusSpec defines the desired state of Nexus.
type NexusSpec struct {
	// Secret is the name of the k8s object Secret related to nexus.
	// Secret should contain a user field with a nexus username and a password field with a nexus password.
	Secret string `json:"secret"`

	// Url is the url of nexus instance.
	Url string `json:"url"`
}

// NexusStatus defines the observed state of Nexus.
type NexusStatus struct {
	// Error represents error message if something went wrong.
	// +optional
	Error string `json:"error,omitempty"`

	// Connected shows if operator is connected to nexus.
	// +optional
	Connected bool `json:"connected"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=nexuses
// +kubebuilder:printcolumn:name="Connected",type="boolean",JSONPath=".status.connected",description="Is connected to nexus"

// Nexus is the Schema for the nexus API.
type Nexus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusSpec   `json:"spec,omitempty"`
	Status NexusStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NexusList contains a list of Nexus.
type NexusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nexus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nexus{}, &NexusList{})
}
