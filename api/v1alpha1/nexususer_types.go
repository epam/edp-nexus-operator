package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-nexus-operator/api/common"
)

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

// NexusUserSpec defines the desired state of NexusUser.
type NexusUserSpec struct {
	// ID is the username of the user.
	// ID should be unique across all users.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=512
	// +kubebuilder:example="new-user"
	ID string `json:"id"`

	// FirstName of the user.
	// +required
	// +kubebuilder:example="John"
	FirstName string `json:"firstName"`

	// LastName of the user.
	// +required
	// +kubebuilder:example="Doe"
	LastName string `json:"lastName"`

	// Email is the email address of the user.
	// +required
	// +kubebuilder:validation:MaxLength=254
	// +kubebuilder:example="john.doe@example"
	Email string `json:"email"`

	// Secret is the reference of the k8s object Secret for the user password.
	// Format: $secret-name:secret-key.
	// After updating Secret user password will be updated.
	// +required
	// +kubebuilder:example="$nexus-user-secret:secret-key"
	Secret string `json:"secret"`

	// Status is a status of the user.
	// +optional
	// +kubebuilder:validation:Enum=active;disabled
	// +kubebuilder:default:=active
	// +kubebuilder:example="active"
	Status string `json:"status"`

	// Roles is a list of roles assigned to user.
	// +required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:example={nx-admin}
	Roles []string `json:"roles"`

	// NexusRef is a reference to Nexus custom resource.
	// +required
	NexusRef common.NexusRef `json:"nexusRef"`
}

// NexusUserStatus defines the observed state of NexusUser.
type NexusUserStatus struct {
	// Value is a status of the user.
	// +optional
	Value string `json:"value,omitempty"`

	// Error is an error message if something went wrong.
	// +optional
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NexusUser is the Schema for the nexususers API.
type NexusUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusUserSpec   `json:"spec,omitempty"`
	Status NexusUserStatus `json:"status,omitempty"`
}

func (in *NexusUser) GetNexusRef() common.NexusRef {
	return in.Spec.NexusRef
}

// +kubebuilder:object:root=true

// NexusUserList contains a list of NexusUser.
type NexusUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NexusUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NexusUser{}, &NexusUserList{})
}
