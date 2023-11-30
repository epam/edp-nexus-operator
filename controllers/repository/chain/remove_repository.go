package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type RemoveRepository struct {
	nexusRepositoryApiClient nexus.Repository
}

func NewRemoveRepository(nexusRepositoryApiClient nexus.Repository) *RemoveRepository {
	return &RemoveRepository{nexusRepositoryApiClient: nexusRepositoryApiClient}
}

func (h *RemoveRepository) ServeRequest(ctx context.Context, repository *nexusApi.NexusRepository) error {
	log := ctrl.LoggerFrom(ctx)

	repoData, err := nexus.GetRepoData(&repository.Spec)
	if err != nil {
		return fmt.Errorf("failed to get repository data: %w", err)
	}

	log = log.WithValues("type", repoData.Type, "format", repoData.Format, "name", repoData.Name)

	log.Info("Deleting repository")

	if err = h.nexusRepositoryApiClient.Delete(ctx, repoData.Name); err != nil {
		if nexus.IsErrNotFound(err) {
			log.Info("Repository doesn't exist, skipping removal")
			return nil
		}

		return fmt.Errorf("failed to delete repository: %w", err)
	}

	log.Info("Repository has been deleted")

	return nil
}
