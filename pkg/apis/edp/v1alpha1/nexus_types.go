package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NexusSpec defines the desired state of Nexus
// +k8s:openapi-gen=true
type NexusSpec struct {
	KeycloakSpec KeycloakSpec   `json:"keycloakSpec, omitempty"`
	Image        string         `json:"image"`
	Version      string         `json:"version"`
	BasePath     string         `json:"basePath"`
	Volumes      []NexusVolumes `json:"volumes, omitempty"`
	Users        []NexusUsers   `json:"users, omitempty"`
	EdpSpec      EdpSpec        `json:"edpSpec"`
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

type EdpSpec struct {
	DnsWildcard string `json:"dnsWildcard"`
}

type NexusVolumes struct {
	Name         string `json:"name"`
	StorageClass string `json:"storage_class"`
	Capacity     string `json:"capacity"`
}

type NexusUsers struct {
	Username  string   `json:"username"`
	FirstName string   `json:"first_name, omitempty"`
	LastName  string   `json:"last_name, omitempty"`
	Email     string   `json:"email, omitempty"`
	Roles     []string `json:"roles, omitempty"`
}

// NexusStatus defines the observed state of Nexus
// +k8s:openapi-gen=true
type NexusStatus struct {
	Available       bool      `json:"available, omitempty"`
	LastTimeUpdated time.Time `json:"lastTimeUpdated, omitempty"`
	Status          string    `json:"status, omitempty"`
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

type KeycloakSpec struct {
	Enabled bool   `json:"enabled, omitempty"`
	Url     string `json:"url, omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Nexus is the Schema for the nexus API
// +k8s:openapi-gen=true
type Nexus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusSpec   `json:"spec,omitempty"`
	Status NexusStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NexusList contains a list of Nexus
type NexusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nexus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nexus{}, &NexusList{})
}
