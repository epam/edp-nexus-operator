package v1

import (
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NexusSpec defines the desired state of Nexus
type NexusSpec struct {
	KeycloakSpec     KeycloakSpec                     `json:"keycloakSpec"`
	ImagePullSecrets []coreV1Api.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Image            string                           `json:"image"`
	Version          string                           `json:"version"`
	BasePath         string                           `json:"basePath,omitempty"`
	Volumes          []NexusVolumes                   `json:"volumes"`
	Users            []NexusUsers                     `json:"users,omitempty"`
	EdpSpec          EdpSpec                          `json:"edpSpec,omitempty"`
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
type NexusStatus struct {
	Available       bool        `json:"available,omitempty"`
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
	Status          string      `json:"status,omitempty"`
}

type KeycloakSpec struct {
	Enabled    bool     `json:"enabled"`
	Url        string   `json:"url,omitempty"`
	Realm      string   `json:"realm,omitempty"`
	Roles      []string `json:"roles,omitempty"`
	ProxyImage string   `json:"proxyImage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:resource:path=nexuses

// Nexus is the Schema for the nexuses API
type Nexus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusSpec   `json:"spec,omitempty"`
	Status NexusStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NexusList contains a list of Nexus
type NexusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nexus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nexus{},
		&NexusList{})
}
