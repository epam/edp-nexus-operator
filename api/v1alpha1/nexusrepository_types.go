package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-nexus-operator/api/common"
)

// NexusRepositorySpec defines the desired state of NexusRepository.
// It should contain only one format of repository - go, maven, npm, etc. and only one type - proxy, hosted or group.
type NexusRepositorySpec struct {
	Apt       *AptSpec       `json:"apt,omitempty"`
	Bower     *BowerSpec     `json:"bower,omitempty"`
	Cocoapods *CocoapodsSpec `json:"cocoapods,omitempty"`
	Conan     *ConanSpec     `json:"conan,omitempty"`
	Conda     *CondaSpec     `json:"conda,omitempty"`
	Docker    *DockerSpec    `json:"docker,omitempty"`
	GitLfs    *GitLfsSpec    `json:"gitLfs,omitempty"`
	Go        *GoSpec        `json:"go,omitempty"`
	Helm      *HelmSpec      `json:"helm,omitempty"`
	Maven     *MavenSpec     `json:"maven,omitempty"`
	Npm       *NpmSpec       `json:"npm,omitempty"`
	Nuget     *NugetSpec     `json:"nuget,omitempty"`
	P2        *P2Spec        `json:"p2,omitempty"`
	Pypi      *PypiSpec      `json:"pypi,omitempty"`
	R         *RSpec         `json:"r,omitempty"`
	Raw       *RawSpec       `json:"raw,omitempty"`
	RubyGems  *RubyGemsSpec  `json:"rubyGems,omitempty"`
	Yum       *YumSpec       `json:"yum,omitempty"`

	// NexusRef is a reference to Nexus custom resource.
	// +required
	NexusRef common.NexusRef `json:"nexusRef"`
}

type NpmSpec struct {
	Group  *NpmGroupRepository  `json:"group,omitempty"`
	Proxy  *NpmProxyRepository  `json:"proxy,omitempty"`
	Hosted *NpmHostedRepository `json:"hosted,omitempty"`
}

type AptSpec struct {
	Proxy  *AptProxyRepository  `json:"proxy,omitempty"`
	Hosted *AptHostedRepository `json:"hosted,omitempty"`
}

type BowerSpec struct {
	Group  *BowerGroupRepository  `json:"group,omitempty"`
	Proxy  *BowerProxyRepository  `json:"proxy,omitempty"`
	Hosted *BowerHostedRepository `json:"hosted,omitempty"`
}

type CocoapodsSpec struct {
	Proxy *CocoapodsProxyRepository `json:"proxy,omitempty"`
}

type ConanSpec struct {
	Proxy *ConanProxyRepository `json:"proxy,omitempty"`
}

type CondaSpec struct {
	Proxy *CondaProxyRepository `json:"proxy,omitempty"`
}

type DockerSpec struct {
	Group  *DockerGroupRepository  `json:"group,omitempty"`
	Proxy  *DockerProxyRepository  `json:"proxy,omitempty"`
	Hosted *DockerHostedRepository `json:"hosted,omitempty"`
}

type GitLfsSpec struct {
	Hosted *GitLfsHostedRepository `json:"hosted,omitempty"`
}

type GoSpec struct {
	Group *GoGroupRepository `json:"group,omitempty"`
	Proxy *GoProxyRepository `json:"proxy,omitempty"`
}

type HelmSpec struct {
	Proxy  *HelmProxyRepository  `json:"proxy,omitempty"`
	Hosted *HelmHostedRepository `json:"hosted,omitempty"`
}

type MavenSpec struct {
	Group  *MavenGroupRepository  `json:"group,omitempty"`
	Proxy  *MavenProxyRepository  `json:"proxy,omitempty"`
	Hosted *MavenHostedRepository `json:"hosted,omitempty"`
}

type NugetSpec struct {
	Group  *NugetGroupRepository  `json:"group,omitempty"`
	Proxy  *NugetProxyRepository  `json:"proxy,omitempty"`
	Hosted *NugetHostedRepository `json:"hosted,omitempty"`
}

type P2Spec struct {
	Proxy *P2ProxyRepository `json:"proxy,omitempty"`
}

type PypiSpec struct {
	Group  *PypiGroupRepository  `json:"group,omitempty"`
	Proxy  *PypiProxyRepository  `json:"proxy,omitempty"`
	Hosted *PypiHostedRepository `json:"hosted,omitempty"`
}

type RSpec struct {
	Group  *RGroupRepository  `json:"group,omitempty"`
	Proxy  *RProxyRepository  `json:"proxy,omitempty"`
	Hosted *RHostedRepository `json:"hosted,omitempty"`
}

type RawSpec struct {
	Group  *RawGroupRepository  `json:"group,omitempty"`
	Proxy  *RawProxyRepository  `json:"proxy,omitempty"`
	Hosted *RawHostedRepository `json:"hosted,omitempty"`
}

type RubyGemsSpec struct {
	Group  *RubyGemsGroupRepository  `json:"group,omitempty"`
	Proxy  *RubyGemsProxyRepository  `json:"proxy,omitempty"`
	Hosted *RubyGemsHostedRepository `json:"hosted,omitempty"`
}

type YumSpec struct {
	Group  *YumGroupRepository  `json:"group,omitempty"`
	Proxy  *YumProxyRepository  `json:"proxy,omitempty"`
	Hosted *YumHostedRepository `json:"hosted,omitempty"`
}

// NexusRepositoryStatus defines the observed state of NexusRepository.
type NexusRepositoryStatus struct {
	// Value is a status of the repository.
	// +optional
	Value string `json:"value,omitempty"`

	// Error is an error message if something went wrong.
	// +optional
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NexusRepository is the Schema for the nexusrepositories API.
type NexusRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NexusRepositorySpec   `json:"spec,omitempty"`
	Status NexusRepositoryStatus `json:"status,omitempty"`
}

func (in *NexusRepository) GetNexusRef() common.NexusRef {
	return in.Spec.NexusRef
}

// +kubebuilder:object:root=true

// NexusRepositoryList contains a list of NexusRepository.
type NexusRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NexusRepository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NexusRepository{}, &NexusRepositoryList{})
}
