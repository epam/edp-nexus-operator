package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-nexus-operator/api/common"
)

const (
	SoftQuotaSpaceRemainingQuota = "spaceRemainingQuota"
	SoftQuotaSpaceUsedQuota      = "spaceUsedQuota"
)

// NexusBlobStoreSpec defines the desired state of NexusBlobStore.
type NexusBlobStoreSpec struct {
	// Name of the BlobStore.
	// Name should be unique across all BlobStores.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Name string `json:"name"`

	// Settings to control the soft quota.
	// +optional
	SoftQuota *SoftQuota `json:"softQuota,omitempty"`

	// File type blobstore.
	// +required
	File File `json:"file,omitempty"`

	// NexusRef is a reference to Nexus custom resource.
	// +required
	NexusRef common.NexusRef `json:"nexusRef"`
}

type File struct {
	// The path to the blobstore contents.
	// This can be an absolute path to anywhere on the system Nexus Repository Manager has access to it or can be a path relative to the sonatype-work directory.
	// +required
	Path string `json:"path,omitempty"`
}

type SoftQuota struct {
	// Type of the soft quota.
	// +required
	// +kubebuilder:validation:Enum=spaceRemainingQuota;spaceUsedQuota
	Type string `json:"type,omitempty"`

	// The limit in MB.
	// +required
	// +kubebuilder:validation:Minimum=1
	Limit int64 `json:"limit,omitempty"`
}

// NexusBlobStoreStatus defines the observed state of NexusBlobStore.
type NexusBlobStoreStatus struct {
	// Value is a status of the blob store.
	// +optional
	Value string `json:"value,omitempty"`

	// Error is an error message if something went wrong.
	// +optional
	Error string `json:"error,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Status of the blob store"

// NexusBlobStore is the Schema for the nexusblobstores API.
type NexusBlobStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusBlobStoreSpec   `json:"spec,omitempty"`
	Status NexusBlobStoreStatus `json:"status,omitempty"`
}

func (in *NexusBlobStore) GetNexusRef() common.NexusRef {
	return in.Spec.NexusRef
}

//+kubebuilder:object:root=true

// NexusBlobStoreList contains a list of NexusBlobStore.
type NexusBlobStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NexusBlobStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NexusBlobStore{}, &NexusBlobStoreList{})
}
