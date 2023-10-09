package chain

import (
	"context"
	"fmt"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

// CreateRole is a handler for creating role.
type CreateRole struct {
	nexusRoleApiClient nexus.Role
}

// NewCreateRole creates an instance of CreateRole handler.
func NewCreateRole(nexusRoleApiClient nexus.Role) *CreateRole {
	return &CreateRole{nexusRoleApiClient: nexusRoleApiClient}
}

// ServeRequest implements the logic of creating role.
func (c CreateRole) ServeRequest(ctx context.Context, role *nexusApi.NexusRole) error {
	log := ctrl.LoggerFrom(ctx).WithValues("id", role.Spec.ID)
	log.Info("Start creating role")

	nexusRole, err := c.nexusRoleApiClient.Get(role.Spec.ID)
	if err != nil {
		if !nexus.IsErrNotFound(err) {
			return fmt.Errorf("failed to get role: %w", err)
		}

		log.Info("Role doesn't exist, creating new one")

		if err = c.nexusRoleApiClient.Create(specToRole(&role.Spec)); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}

		log.Info("Role has been created")

		return nil
	}

	if roleChanged(&role.Spec, nexusRole) {
		log.Info("Updating role")

		if err = c.nexusRoleApiClient.Update(role.Spec.ID, specToRole(&role.Spec)); err != nil {
			return fmt.Errorf("failed to update role: %w", err)
		}

		log.Info("Role has been updated")
	}

	return nil
}

func roleChanged(spec *nexusApi.NexusRoleSpec, nexusRole *security.Role) bool {
	if spec.Description != nexusRole.Description ||
		spec.Name != nexusRole.Name ||
		!slices.Equal(spec.Privileges, nexusRole.Privileges) {
		return true
	}

	return false
}

func specToRole(spec *nexusApi.NexusRoleSpec) security.Role {
	return security.Role{
		ID:          spec.ID,
		Name:        spec.Name,
		Description: spec.Description,
		Privileges:  spec.Privileges,
	}
}
