// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha1 contains API Schema definitions for the edp v1alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=v2.edp.epam.com
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func RegisterTypes(sch *runtime.Scheme) {
	sch.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"},
		&Nexus{},
		&NexusUser{},
		&NexusList{},
		&NexusUserList{})
}
