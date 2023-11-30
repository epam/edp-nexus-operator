package v1alpha1

type ProxySpec struct {
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
	HTTPClient `json:"httpClient"`

	// The name of the routing rule assigned to this repository.
	// +optional
	// +kubebuilder:example=go-proxy-routing-rule
	RoutingRule *string `json:"routingRule,omitempty"`

	*Cleanup `json:"cleanup,omitempty"`
}

type HostedSpec struct {
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
	Storage HostedStorage `json:"storage"`

	*Cleanup   `json:"cleanup,omitempty"`
	*Component `json:"component,omitempty"`
}

type GroupSpec struct {
	// A unique identifier for this repository.
	// Only letters, digits, underscores(_), hyphens(-), and dots(.) are allowed and may not start with underscore or dot.
	// +required
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`
	Name string `json:"name"`

	// Online determines if the repository accepts incoming requests.
	// +optional
	// +kubebuilder:default=true
	Online bool `json:"online"`

	// Group configuration.
	// +required
	Group `json:"group"`

	// Storage configuration.
	// +optional
	// +kubebuilder:default={"blobStoreName":"default","strictContentTypeValidation":true}
	Storage `json:"storage"`
}

// Group contains repository group configuration data.
type Group struct {
	// Member repositories' names.
	// +required
	MemberNames []string `json:"memberNames"`
}

// GroupDeploy contains repository group deployment configuration data.
type GroupDeploy struct {
	// Member repositories' names.
	// +required
	MemberNames []string `json:"memberNames"`
	// Pro-only: This field is for the Group Deployment feature available in NXRM Pro.
	// +optional
	WritableMember *string `json:"writableMember,omitempty"`
}

// HTTPClient contains HTTP client configuration data.
type HTTPClient struct {
	Authentication *HTTPClientAuthentication `json:"authentication,omitempty"`

	// Auto-block outbound connections on the repository if remote peer is detected as unreachable/unresponsive
	// +optional
	// +kubebuilder:default=true
	AutoBlock bool `json:"autoBlock"`

	// Block outbound connections on the repository.
	// +optional
	Blocked    bool                  `json:"blocked"`
	Connection *HTTPClientConnection `json:"connection,omitempty"`
}

// HTTPClientWithPreemptiveAuth contains HTTP client configuration data.
type HTTPClientWithPreemptiveAuth struct {
	// Auto-block outbound connections on the repository if remote peer is detected as unreachable/unresponsive
	// +optional
	// +kubebuilder:default=true
	AutoBlock bool `json:"autoBlock"`

	// Whether to block outbound connections on the repository.
	// +optional
	Blocked bool `json:"blocked"`

	Authentication *HTTPClientAuthenticationWithPreemptive `json:"authentication,omitempty"`
	Connection     *HTTPClientConnection                   `json:"connection,omitempty"`
}

// HTTPClientConnection contains HTTP client connection configuration data.
type HTTPClientConnection struct {
	// Whether to enable redirects to the same location (required by some servers)
	EnableCircularRedirects *bool `json:"enableCircularRedirects,omitempty"`
	// Whether to allow cookies to be stored and used
	EnableCookies *bool `json:"enableCookies,omitempty"`
	// Total retries if the initial connection attempt suffers a timeout
	Retries *int `json:"retries,omitempty"`
	// Seconds to wait for activity before stopping and retrying the connection",
	Timeout *int `json:"timeout,omitempty"`
	// Custom fragment to append to User-Agent header in HTTP requests
	UserAgentSuffix string `json:"userAgentSuffix,omitempty"`
	// Use certificates stored in the Nexus Repository Manager truststore to connect to external systems
	UseTrustStore *bool `json:"useTrustStore,omitempty"`
}

// HTTPClientAuthentication contains HTTP client authentication configuration data.
type HTTPClientAuthentication struct {
	NTLMDomain string `json:"ntlmDomain,omitempty"`
	NTLMHost   string `json:"ntlmHost,omitempty"`

	// Type of authentication to use.
	// +optional
	// +kubebuilder:default=username
	// +kubebuilder:validation:Enum=username;ntlm
	Type string `json:"type"`

	// Password for authentication.
	// +required
	Password string `json:"password,omitempty"`

	// Username for authentication.
	// +required
	Username string `json:"username,omitempty"`
}

// HTTPClientAuthenticationWithPreemptive contains HTTP client authentication configuration data.
type HTTPClientAuthenticationWithPreemptive struct {
	NTLMDomain string `json:"ntlmDomain,omitempty"`
	NTLMHost   string `json:"ntlmHost,omitempty"`
	Password   string `json:"password,omitempty"`

	// Type of authentication to use.
	// +optional
	// +kubebuilder:default=username
	// +kubebuilder:validation:Enum=username;ntlm
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	// Whether to use pre-emptive authentication. Use with caution. Defaults to false.
	Preemptive *bool `json:"preemptive,omitempty"`
}

type Cleanup struct {
	//  Components that match any of the applied policies will be deleted.
	// +required
	PolicyNames []string `json:"policyNames"`
}

type NegativeCache struct {
	// Whether to cache responses for content not present in the proxied repository.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// How long to cache the fact that a file was not found in the repository (in minutes).
	// +optional
	// +kubebuilder:default=1440
	TTL int `json:"timeToLive"`
}

// Proxy contains Proxy Repository data.
type Proxy struct {
	// How long to cache artifacts before rechecking the remote repository (in minutes)
	// +optional
	// +kubebuilder:default=1440
	ContentMaxAge int `json:"contentMaxAge"`

	// How long to cache metadata before rechecking the remote repository (in minutes)
	// +optional
	// +kubebuilder:default=1440
	MetadataMaxAge int `json:"metadataMaxAge"`

	// Location of the remote repository being proxied.
	// +required
	// +kubebuilder:example=`https://remote-repository.com`
	RemoteURL string `json:"remoteUrl"`
}

type Component struct {
	// Components in this repository count as proprietary for namespace conflict attacks (requires Sonatype Nexus Firewall)
	ProprietaryComponents bool `json:"proprietaryComponents"`
}

// HostedStorage contains repository storage for hosted.
type HostedStorage struct {
	// Blob store used to store repository contents.
	// +optional
	// +kubebuilder:default=default
	// +kubebuilder:example=default
	BlobStoreName string `json:"blobStoreName"`

	// StrictContentTypeValidation: Whether to validate uploaded content's MIME type appropriate for the repository format.
	// +optional
	// +kubebuilder:default=true
	StrictContentTypeValidation bool `json:"strictContentTypeValidation"`

	// WritePolicy controls if deployments of and updates to assets are allowed.
	// +optional
	// +kubebuilder:default=ALLOW_ONCE
	// +kubebuilder:validation:Enum=ALLOW;ALLOW_ONCE;DENY;REPLICATION_ONLY
	WritePolicy string `json:"writePolicy,omitempty"`
}

// Storage contains repository storage.
type Storage struct {
	// Blob store used to store repository contents.
	// +optional
	// +kubebuilder:default=default
	// +kubebuilder:example=default
	BlobStoreName string `json:"blobStoreName"`

	// StrictContentTypeValidation: Whether to validate uploaded content's MIME type appropriate for the repository format.
	// +optional
	// +kubebuilder:default=true
	StrictContentTypeValidation bool `json:"strictContentTypeValidation"`
}
