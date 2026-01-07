package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-nexus-operator/api/common"
)

// NexusCleanupPolicySpec defines the desired state of NexusCleanupPolicy.
type NexusCleanupPolicySpec struct {
	// Name is a unique name for the cleanup policy.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=512
	// +required
	// +kubebuilder:example="go-cleanup-policy"
	Name string `json:"name"`

	// Format that this cleanup policy can be applied to.
	// +required
	// +kubebuilder:example="go"
	// +kubebuilder:validation:Enum=apt;bower;cocoapods;conan;conda;docker;gitlfs;go;helm;maven2;npm;nuget;p2;pypi;r;raw;rubygems;yum
	Format string `json:"format"`

	// Description of the cleanup policy.
	// +optional
	// +kubebuilder:example="Cleanup policy for go format"
	Description string `json:"description,omitempty"`

	// Criteria for the cleanup policy.
	// +required
	Criteria Criteria `json:"criteria"`

	// NexusRef is a reference to Nexus custom resource.
	// +required
	NexusRef common.NexusRef `json:"nexusRef"`
}

type Criteria struct {
	// ReleaseType removes components that are of the following release type.
	// +optional
	// +kubebuilder:example="RELEASES"
	// +kubebuilder:validation:Enum=RELEASES;PRERELEASES;""
	ReleaseType string `json:"releaseType,omitempty"`

	// LastBlobUpdated removes components published over “x” days ago.
	// +optional
	// +kubebuilder:example="30"
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=24855
	LastBlobUpdated int `json:"lastBlobUpdated,omitempty"`

	// LastDownloaded removes components downloaded over “x” days.
	// +optional
	// +kubebuilder:example="30"
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=24855
	LastDownloaded int `json:"lastDownloaded,omitempty"`

	// AssetRegex removes components that match the given regex.
	// +optional
	// +kubebuilder:example=".*"
	AssetRegex string `json:"assetRegex,omitempty"`
}

// NexusCleanupPolicyStatus defines the observed state of NexusCleanupPolicy.
type NexusCleanupPolicyStatus struct {
	// Value is a status of the cleanup policy.
	// +optional
	Value string `json:"value,omitempty"`

	// Error is an error message if something went wrong.
	// +optional
	Error string `json:"error,omitempty"`
}

func (in *NexusCleanupPolicy) GetNexusRef() common.NexusRef {
	return in.Spec.NexusRef
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NexusCleanupPolicy is the Schema for the cleanuppolicies API.
type NexusCleanupPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusCleanupPolicySpec   `json:"spec,omitempty"`
	Status NexusCleanupPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NexusCleanupPolicyList contains a list of NexusCleanupPolicy.
type NexusCleanupPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NexusCleanupPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NexusCleanupPolicy{}, &NexusCleanupPolicyList{})
}
