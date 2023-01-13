package v1alpha1

import (
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NexusSpec defines the desired state of Nexus
// +k8s:openapi-gen=true
type NexusSpec struct {
	KeycloakSpec KeycloakSpec `json:"keycloakSpec"`
	// +nullable
	ImagePullSecrets []coreV1Api.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Image            string                           `json:"image"`
	Version          string                           `json:"version"`
	BasePath         string                           `json:"basePath,omitempty"`
	Volumes          []NexusVolumes                   `json:"volumes"`
	Users            []NexusUsers                     `json:"users,omitempty"`
	EdpSpec          EdpSpec                          `json:"edpSpec,omitempty"`
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

type EdpSpec struct {
	DnsWildcard string `json:"dnsWildcard,omitempty"`
}

type NexusVolumes struct {
	Name         string `json:"name,omitempty"`
	StorageClass string `json:"storage_class,omitempty"`
	Capacity     string `json:"capacity,omitempty"`
}

type NexusUsers struct {
	Username  string   `json:"username,omitempty"`
	FirstName string   `json:"first_name,omitempty"`
	LastName  string   `json:"last_name,omitempty"`
	Email     string   `json:"email,omitempty"`
	Roles     []string `json:"roles,omitempty"`
}

// NexusStatus defines the observed state of Nexus
// +k8s:openapi-gen=true
type NexusStatus struct {
	Available       bool        `json:"available,omitempty"`
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
	Status          string      `json:"status,omitempty"`
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

type KeycloakSpec struct {
	Enabled    bool     `json:"enabled"`
	Url        string   `json:"url,omitempty"`
	Realm      string   `json:"realm,omitempty"`
	Roles      []string `json:"roles,omitempty"`
	ProxyImage string   `json:"proxyImage,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Nexus is the Schema for the nexus API
// +k8s:openapi-gen=true
// +kubebuilder:deprecatedversion
type Nexus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusSpec   `json:"spec,omitempty"`
	Status NexusStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NexusList contains a list of Nexus.
type NexusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nexus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nexus{},
		&NexusList{})
}
