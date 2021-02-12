package controller

import (
	"github.com/epam/edp-nexus-operator/v2/pkg/controller/nexus"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, nexus.Add)
}
