package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NexusUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NexusUserSpec   `json:"spec,omitempty"`
	Status            NexusUserStatus `json:"status,omitempty"`
}

type NexusUserSpec struct {
	OwnerName string   `json:"ownerName"`
	UserID    string   `json:"userId"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Status    string   `json:"status"`
}

func (in NexusUser) OwnerName() string {
	return in.Spec.OwnerName
}

type NexusUserStatus struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NexusUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NexusUser `json:"items"`
}
