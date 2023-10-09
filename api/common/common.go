package common

// StatusCreated is success status for Nexus resources.
const StatusCreated = "created"

// NexusRef is a reference to a Nexus instance.
type NexusRef struct {
	// Kind specifies the kind of the Nexus resource.
	// +optional
	// +kubebuilder:default=Nexus
	Kind string `json:"kind"`

	// Name specifies the name of the Nexus resource.
	// +required
	Name string `json:"name"`
}

type HasNexusRef interface {
	GetNexusRef() NexusRef
}
