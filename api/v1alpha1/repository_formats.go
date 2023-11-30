package v1alpha1

type AptHostedRepository struct {
	HostedSpec `json:",inline"`

	Apt        AptHosted `json:"apt"`
	AptSigning `json:"aptSigning"`
}

type AptProxyRepository struct {
	ProxySpec `json:",inline"`

	// Apt configuration.
	// +required
	Apt AptProxy `json:"apt"`
}

// Apt contains data of proxy repositories of format Apt.
type AptProxy struct {
	// Distribution to fetch.
	// +required
	Distribution string `json:"distribution"`

	// Whether this repository is flat.
	// +optional
	// +kubebuilder:default=false
	Flat bool `json:"flat"`
}

// Apt contains data of hosted repositories of format Apt.
type AptHosted struct {
	// Distribution to fetch
	Distribution string `json:"distribution"`
}

// AptSigning contains signing data of hosted repositores of format Apt.
type AptSigning struct {
	// PGP signing key pair (armored private key e.g. gpg --export-secret-key --armor)
	Keypair string `json:"keypair"`
	// Passphrase to access PGP signing key
	Passphrase *string `json:"passphrase,omitempty"`
}

type BowerGroupRepository struct {
	GroupSpec `json:",inline"`
}

type BowerHostedRepository struct {
	HostedSpec `json:",inline"`
}

type BowerProxyRepository struct {
	ProxySpec `json:",inline"`

	Bower `json:"bower"`
}

type Bower struct {
	// Whether to force Bower to retrieve packages through this proxy repository
	RewritePackageUrls bool `json:"rewritePackageUrls"`
}

type CocoapodsProxyRepository struct {
	ProxySpec `json:",inline"`
}

type ConanProxyRepository struct {
	ProxySpec `json:",inline"`
}

type CondaProxyRepository struct {
	ProxySpec `json:",inline"`
}

type DockerGroupRepository struct {
	// A unique identifier for this repository.
	// Only letters, digits, underscores(_), hyphens(-), and dots(.) are allowed and may not start with underscore or dot.
	// +required
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`
	Name string `json:"name"`

	// Online determines if the repository accepts incoming requests.
	// +optional
	// +kubebuilder:default=true
	Online bool `json:"online"`

	// Storage configuration.
	// +optional
	// +kubebuilder:default={"blobStoreName":"default","strictContentTypeValidation":true}
	Storage `json:"storage"`

	// Group configuration.
	// +optional
	Group GroupDeploy `json:"group"`

	Docker `json:"docker"`
}

type DockerHostedRepository struct {
	HostedSpec `json:",inline"`

	Docker `json:"docker"`
}

type DockerProxyRepository struct {
	ProxySpec `json:",inline"`

	Docker      `json:"docker"`
	DockerProxy `json:"dockerProxy"`
}

// Docker contains data of a Docker Repositoriy.
type Docker struct {
	// Whether to force authentication (Docker Bearer Token Realm required if false)
	ForceBasicAuth bool `json:"forceBasicAuth"`
	// Create an HTTP connector at specified port
	HTTPPort *int `json:"httpPort,omitempty"`
	// Create an HTTPS connector at specified port
	HTTPSPort *int `json:"httpsPort,omitempty"`
	// Whether to allow clients to use the V1 API to interact with this repository
	V1Enabled bool `json:"v1Enabled"`
}

// DockerProxy contains data of a Docker Proxy Repository.
type DockerProxy struct {
	// Type of Docker Index.
	// +optional
	// +kubebuilder:default=REGISTRY
	// +kubebuilder:validation:Enum=HUB;REGISTRY;CUSTOM
	IndexType string `json:"indexType"`

	// Url of Docker Index to use.
	// +optional
	// TODO: add cel validation. (Required if indexType is CUSTOM)
	IndexURL *string `json:"indexUrl,omitempty"`
}

type GitLfsHostedRepository struct {
	HostedSpec `json:",inline"`
}

type GoGroupRepository struct {
	GroupSpec `json:",inline"`
}

type GoProxyRepository struct {
	ProxySpec `json:",inline"`
}

type HelmHostedRepository struct {
	HostedSpec `json:",inline"`
}

type HelmProxyRepository struct {
	ProxySpec `json:",inline"`
}

// Validate that all paths are maven artifact or metadata paths.
type MavenLayoutPolicy string

// Content Disposition.
type MavenContentDisposition string

type MavenGroupRepository struct {
	GroupSpec `json:",inline"`
}

type MavenHostedRepository struct {
	Maven `json:"maven"`

	HostedSpec `json:",inline"`
}

type MavenProxyRepository struct {
	// A unique identifier for this repository.
	// Only letters, digits, underscores(_), hyphens(-), and dots(.) are allowed and may not start with underscore or dot.
	// +required
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`
	Name string `json:"name"`

	// Online determines if the repository accepts incoming requests.
	// +optional
	// +kubebuilder:default=true
	Online bool `json:"online"`

	// Storage configuration.
	// +optional
	// +kubebuilder:default={"blobStoreName":"default","strictContentTypeValidation":true}
	Storage `json:"storage"`

	// Proxy configuration.
	// +required
	Proxy `json:"proxy"`

	// Negative cache configuration.
	// +optional
	// +kubebuilder:default={"enabled":true,"timeToLive":1440}
	NegativeCache `json:"negativeCache"`

	// HTTP client configuration.
	// +optional
	// +kubebuilder:default={"autoBlock":true}
	HTTPClient HTTPClientWithPreemptiveAuth `json:"httpClient"`

	// The name of the routing rule assigned to this repository.
	// +optional
	// +kubebuilder:example=go-proxy-routing-rule
	RoutingRule *string `json:"routingRule,omitempty"`

	*Cleanup `json:"cleanup,omitempty"`

	// Maven contains additional data of maven repository.
	// +optional
	// +kubebuilder:default={"versionPolicy":"RELEASE","layoutPolicy":"STRICT","contentDisposition":"INLINE"}
	Maven `json:"maven"`
}

// Maven contains additional data of maven repository.
type Maven struct {
	// VersionPolicy is a type of artifact that this repository stores.
	// +optional
	// +kubebuilder:default=RELEASE
	// +kubebuilder:validation:Enum=RELEASE;SNAPSHOT;MIXED
	VersionPolicy string `json:"versionPolicy"`

	// Validate that all paths are maven artifact or metadata paths.
	// +optional
	// +kubebuilder:default=STRICT
	// +kubebuilder:validation:Enum=STRICT;PERMISSIVE
	LayoutPolicy string `json:"layoutPolicy"`

	// Add Content-Disposition header as 'Attachment' to disable some content from being inline in a browser.
	// +optional
	// +kubebuilder:default=INLINE
	// +kubebuilder:validation:Enum=INLINE;ATTACHMENT
	ContentDisposition string `json:"contentDisposition"`
}

type NpmGroupRepository struct {
	GroupSpec `json:",inline"`
}

type NpmHostedRepository struct {
	HostedSpec `json:",inline"`
}

type NpmProxyRepository struct {
	ProxySpec `json:",inline"`

	*Npm `json:"npm,omitempty"`
}

type Npm struct {
	// Remove Non-Cataloged Versions
	RemoveNonCataloged bool `json:"removeNonCataloged"`
	// Remove Quarantined Versions
	RemoveQuarantined bool `json:"removeQuarantined"`
}

type NugetGroupRepository struct {
	GroupSpec `json:",inline"`
}

type NugetHostedRepository struct {
	HostedSpec `json:",inline"`
}

type NugetProxyRepository struct {
	ProxySpec `json:",inline"`

	NugetProxy `json:"nugetProxy"`
}

// NugetProxy contains data specific to proxy repositories of format Nuget.
type NugetProxy struct {
	// How long to cache query results from the proxied repository (in seconds)
	// +optional
	// +kubebuilder:default=3600
	QueryCacheItemMaxAge int `json:"queryCacheItemMaxAge"`
	// NugetVersion is the used Nuget protocol version.
	// +optional
	// +kubebuilder:default=V3
	// +kubebuilder:validation:Enum=V2;V3
	NugetVersion string `json:"nugetVersion"`
}

type P2ProxyRepository struct {
	ProxySpec `json:",inline"`
}

type PypiGroupRepository struct {
	GroupSpec `json:",inline"`
}

type PypiHostedRepository struct {
	HostedSpec `json:",inline"`
}

type PypiProxyRepository struct {
	ProxySpec `json:",inline"`
}

type RGroupRepository struct {
	GroupSpec `json:",inline"`
}

type RHostedRepository struct {
	HostedSpec `json:",inline"`
}

type RProxyRepository struct {
	ProxySpec `json:",inline"`
}

type RawGroupRepository struct {
	GroupSpec `json:",inline"`

	*Raw `json:"raw,omitempty"`
}

type RawHostedRepository struct {
	HostedSpec `json:",inline"`

	*Raw `json:"raw,omitempty"`
}

type RawProxyRepository struct {
	ProxySpec `json:",inline"`
	*Raw      `json:"raw,omitempty"`
}

type Raw struct {
	// +optional
	// +kubebuilder:validation:Enum=INLINE;ATTACHMENT
	// TODO: check default value
	ContentDisposition *string `json:"contentDisposition,omitempty"`
}

type RubyGemsGroupRepository struct {
	GroupSpec `json:",inline"`
}

type RubyGemsHostedRepository struct {
	HostedSpec `json:",inline"`
}

type RubyGemsProxyRepository struct {
	ProxySpec `json:",inline"`
}

type YumGroupRepository struct {
	GroupSpec `json:",inline"`

	*YumSigning `json:"yumSigning,omitempty"`
}

type YumHostedRepository struct {
	HostedSpec `json:",inline"`

	Yum `json:"yum"`
}

type YumProxyRepository struct {
	ProxySpec   `json:",inline"`
	*YumSigning `json:"yumSigning,omitempty"`
}

// Yum contains data of hosted repositories of format Yum.
type Yum struct {
	RepodataDepth int `json:"repodataDepth"`

	// +optional
	// +kubebuilder:validation:Enum=PERMISSIVE;STRICT
	// TODO: check default value
	DeployPolicy *string `json:"deployPolicy,omitempty"`
}

type YumSigning struct {
	// PGP signing key pair (armored private key e.g. gpg --export-secret-key --armor)
	Keypair *string `json:"keypair,omitempty"`
	// Passphrase to access PGP signing key
	Passphrase *string `json:"passphrase,omitempty"`
}
