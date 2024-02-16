package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type CreateNexusCleanupPolicy struct {
	apiClient nexus.NexusCleanupPolicyManager
}

func NewCreateNexusCleanupPolicy(apiClient nexus.NexusCleanupPolicyManager) *CreateNexusCleanupPolicy {
	return &CreateNexusCleanupPolicy{apiClient: apiClient}
}

func (c *CreateNexusCleanupPolicy) ServeRequest(ctx context.Context, policy *nexusApi.NexusCleanupPolicy) error {
	log := ctrl.LoggerFrom(ctx).WithValues("name", policy.Spec.Name)
	log.Info("Start creating cleanup policy")

	_, err := c.apiClient.Get(ctx, policy.Spec.Name)
	if err != nil {
		if !nexus.IsErrNotFound(err) {
			return fmt.Errorf("failed to get cleanup policy: %w", err)
		}

		log.Info("Cleanup policy doesn't exist, creating new one")

		if err = c.apiClient.Create(ctx, specToCleanupPolicy(&policy.Spec)); err != nil {
			return fmt.Errorf("failed to create cleanup policy: %w", err)
		}

		log.Info("Cleanup policy has been created")

		return nil
	}

	log.Info("Updating cleanup policy")

	if err = c.apiClient.Update(ctx, policy.Spec.Name, specToCleanupPolicy(&policy.Spec)); err != nil {
		return fmt.Errorf("failed to update cleanup policy: %w", err)
	}

	log.Info("Cleanup policy has been updated")

	return nil
}

func specToCleanupPolicy(spec *nexusApi.NexusCleanupPolicySpec) *nexus.NexusCleanupPolicy {
	p := &nexus.NexusCleanupPolicy{
		Name:   spec.Name,
		Format: spec.Format,
		Notes:  spec.Description,
	}

	releaseType := spec.Criteria.ReleaseType
	if releaseType != "" {
		p.CriteriaReleaseType = &releaseType
	}

	lastDownloaded := spec.Criteria.LastDownloaded
	if lastDownloaded != 0 {
		p.CriteriaLastDownloaded = ptr.To(lastDownloaded)
	}

	lastBlobUpdated := spec.Criteria.LastBlobUpdated
	if lastBlobUpdated != 0 {
		p.CriteriaLastBlobUpdated = ptr.To(lastBlobUpdated)
	}

	assetRegex := spec.Criteria.AssetRegex
	if assetRegex != "" {
		p.CriteriaAssetRegex = &assetRegex
	}

	return p
}
