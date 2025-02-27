package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

// RemoveRole is a handler for removing role.
type RemoveRole struct {
	nexusRoleApiClient nexus.Role
}

// NewRemoveRole creates an instance of RemoveRole handler.
func NewRemoveRole(nexusRoleApiClient nexus.Role) *RemoveRole {
	return &RemoveRole{nexusRoleApiClient: nexusRoleApiClient}
}

// ServeRequest implements the logic of removing role.
func (c RemoveRole) ServeRequest(ctx context.Context, role *nexusApi.NexusRole) error {
	log := ctrl.LoggerFrom(ctx).WithValues("id", role.Spec.ID)
	log.Info("Start removing role")

	if err := c.nexusRoleApiClient.Delete(role.Spec.ID); err != nil {
		if !nexus.IsErrNotFound(err) {
			return fmt.Errorf("failed to delete role: %w", err)
		}

		log.Info("Role doesn't exist, skipping removal")
	}

	log.Info("Role has been removed")

	return nil
}
