package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

// RemoveUser is a handler for removing user.
type RemoveUser struct {
	nexusUserApiClient nexus.User
}

// NewRemoveUser creates an instance of RemoveUser handler.
func NewRemoveUser(nexusUserApiClient nexus.User) *RemoveUser {
	return &RemoveUser{nexusUserApiClient: nexusUserApiClient}
}

// ServeRequest implements the logic of removing user.
func (c RemoveUser) ServeRequest(ctx context.Context, user *nexusApi.NexusUser) error {
	log := ctrl.LoggerFrom(ctx).WithValues("id", user.Spec.ID)
	log.Info("Start removing user")

	if err := c.nexusUserApiClient.Delete(user.Spec.ID); err != nil {
		if !nexus.IsErrNotFound(err) {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		log.Info("User doesn't exist, skipping removal")
	}

	log.Info("User has been removed")

	return nil
}
