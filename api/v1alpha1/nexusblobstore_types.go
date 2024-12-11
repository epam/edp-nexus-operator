package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-nexus-operator/api/common"
)

const (
	SoftQuotaSpaceRemainingQuota = "spaceRemainingQuota"
	SoftQuotaSpaceUsedQuota      = "spaceUsedQuota"
	S3BucketEncryptionTypeS3     = "s3ManagedEncryption"
	S3BucketEncryptionTypeKms    = "kmsManagedEncryption"
	S3BucketEncryptionTypeNone   = "none"
	S3SingerTypeDefault          = "DEFAULT"
	S3SingerTypeS3               = "S3SignerType"
	S3SingerTypeAWSS3V4          = "AWSS3V4SignerType"
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
	// +optional
	File *File `json:"file,omitempty"`

	// S3 type blobstore.
	// +optional
	S3 *S3 `json:"s3,omitempty"`

	// NexusRef is a reference to Nexus custom resource.
	// +required
	NexusRef common.NexusRef `json:"nexusRef"`
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

type File struct {
	// The path to the blobstore contents.
	// This can be an absolute path to anywhere on the system Nexus Repository Manager has access to it or can be a path relative to the sonatype-work directory.
	// +required
	Path string `json:"path,omitempty"`
}

type S3 struct {
	// Details of the S3 bucket such as name and region.
	// +required
	Bucket S3Bucket `json:"bucket"`

	// The type of encryption to use if any.
	// +optional
	Encryption *S3Encryption `json:"encryption,omitempty"`

	// Security details for granting access the S3 API.
	// +optional
	BucketSecurity *S3BucketSecurity `json:"bucketSecurity,omitempty"`

	// A custom endpoint URL, signer type and whether path style access is enabled.
	AdvancedBucketConnection *S3AdvancedBucketConnection `json:"advancedBucketConnection,omitempty"`
}

type S3Bucket struct {
	// The AWS region to create a new S3 bucket in or an existing S3 bucket's region.
	// +optional
	// +kubebuilder:default="DEFAULT"
	Region string `json:"region"`

	// The name of the S3 bucket.
	// +required
	Name string `json:"name"`

	// The S3 blob store (i.e. S3 object) key prefix.
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// How many days until deleted blobs are finally removed from the S3 bucket (-1 to disable).
	// +optional
	// +kubebuilder:default=3
	Expiration int32 `json:"expiration"`
}

type S3Encryption struct {
	// The type of S3 server side encryption to use.
	// +optional
	// +kubebuilder:validation:Enum=none;s3ManagedEncryption;kmsManagedEncryption
	Type string `json:"encryptionType,omitempty"`

	// If using KMS encryption, you can supply a Key ID. If left blank, then the default will be used.
	// +optional
	Key string `json:"encryptionKey,omitempty"`
}

type S3BucketSecurity struct {
	// An IAM access key ID for granting access to the S3 bucket.
	// +required
	AccessKeyID common.SourceRef `json:"accessKeyId"`

	// The secret access key associated with the specified IAM access key ID.
	// +required
	SecretAccessKey common.SourceRef `json:"secretAccessKey"`

	// An IAM role to assume in order to access the S3 bucket.
	// +optional
	Role string `json:"role,omitempty"`

	// An AWS STS session token associated with temporary security credentials which grant access to the S3 bucket.
	// +optional
	SessionToken *common.SourceRef `json:"sessionToken,omitempty"`
}

type S3AdvancedBucketConnection struct {
	// A custom endpoint URL for third party object stores using the S3 API.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// An API signature version which may be required for third party object stores using the S3 API.
	// +optional
	// +kubebuilder:validation:Enum=DEFAULT;S3SignerType;AWSS3V4SignerType
	SignerType string `json:"signerType,omitempty"`

	// Setting this flag will result in path-style access being used for all requests.
	// +optional
	// +kubebuilder:default=false
	ForcePathStyle bool `json:"forcePathStyle"`

	// Setting this value will override the default connection pool size of Nexus of the s3 client for this blobstore.
	MaxConnectionPoolSize int32 `json:"maxConnectionPoolSize,omitempty"`
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
