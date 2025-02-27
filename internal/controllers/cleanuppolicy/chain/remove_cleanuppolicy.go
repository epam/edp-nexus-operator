package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type RemoveCleanupPolicy struct {
	apiClient nexus.NexusCleanupPolicyManager
}

func NewRemoveCleanupPolicy(apiClient nexus.NexusCleanupPolicyManager) *RemoveCleanupPolicy {
	return &RemoveCleanupPolicy{apiClient: apiClient}
}

func (c *RemoveCleanupPolicy) ServeRequest(ctx context.Context, policy *nexusApi.NexusCleanupPolicy) error {
	log := ctrl.LoggerFrom(ctx).WithValues("name", policy.Spec.Name)
	log.Info("Start removing cleanup policy")

	if err := c.apiClient.Delete(ctx, policy.Spec.Name); err != nil {
		if !nexus.IsErrNotFound(err) {
			return fmt.Errorf("failed to delete cleanup policy: %w", err)
		}

		log.Info("Cleanup policy doesn't exist, skipping removal")
	}

	log.Info("Cleanup policy has been removed")

	return nil
}
