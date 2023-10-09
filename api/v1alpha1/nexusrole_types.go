package v1alpha1

import (
	"github.com/epam/edp-nexus-operator/api/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NexusRoleSpec defines the desired state of NexusRole.
type NexusRoleSpec struct {
	// ID is the id of the role.
	// ID should be unique across all roles.
	// Do not edit this field after creation. Otherwise, the role will be recreated.
	// +required
	// +kubebuilder:example="nx-admin"
	ID string `json:"id"`

	// Name is the name of the role.
	// +required
	// +kubebuilder:example="nx-admin"
	Name string `json:"name"`

	// Description of nexus role.
	// +optional
	// +kubebuilder:example="Administrator role"
	Description string `json:"description,omitempty"`

	// Privileges is a list of privileges assigned to role.
	// +nullable
	// +optional
	// +kubebuilder:example={nx-all}
	Privileges []string `json:"privileges,omitempty"`

	// NexusRef is a reference to Nexus custom resource.
	// +required
	NexusRef common.NexusRef `json:"nexusRef"`
}

// NexusRoleStatus defines the observed state of NexusRole.
type NexusRoleStatus struct {
	// Value is a status of the group.
	// +optional
	Value string `json:"value,omitempty"`

	// Error is an error message if something went wrong.
	// +optional
	Error string `json:"error,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NexusRole is the Schema for the nexusroles API.
type NexusRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusRoleSpec   `json:"spec,omitempty"`
	Status NexusRoleStatus `json:"status,omitempty"`
}

func (in *NexusRole) GetNexusRef() common.NexusRef {
	return in.Spec.NexusRef
}

//+kubebuilder:object:root=true

// NexusRoleList contains a list of NexusRole.
type NexusRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NexusRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NexusRole{}, &NexusRoleList{})
}
