package chain

import (
	"context"
	"errors"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

// CreateRepository is a handler for creating repository.
type CreateRepository struct {
	nexusRepositoryApiClient nexus.Repository
	k8sClient                client.Client
}

// NewCreateRepository creates an instance of CreateRepository handler.
func NewCreateRepository(nexusRepositoryApiClient nexus.Repository, k8sClient client.Client) *CreateRepository {
	return &CreateRepository{nexusRepositoryApiClient: nexusRepositoryApiClient, k8sClient: k8sClient}
}

// ServeRequest implements the logic of creating repository.
func (c *CreateRepository) ServeRequest(ctx context.Context, repository *nexusApi.NexusRepository) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creating repository")

	repoData, err := getRepoDataWithCredentials(ctx, c.k8sClient, repository.Namespace, &repository.Spec)
	if err != nil {
		return err
	}

	log = log.WithValues("type", repoData.Type, "format", repoData.Format, "name", repoData.Name)

	log.Info("Getting repository")

	_, err = c.nexusRepositoryApiClient.Get(ctx, repoData.Name, repoData.Format, repoData.Type)
	if err != nil {
		if errors.Is(err, nexus.ErrNotFound) {
			log.Info("Repository doesn't exist, creating new one")

			if err = c.nexusRepositoryApiClient.Create(ctx, repoData.Format, repoData.Type, repoData.Data); err != nil {
				return fmt.Errorf("failed to create repository: %w", err)
			}

			log.Info("Repository has been created")

			return nil
		}

		return fmt.Errorf("failed to get repository: %w", err)
	}

	log.Info("Updating repository")

	if err = c.nexusRepositoryApiClient.Update(ctx, repoData.Name, repoData.Format, repoData.Type, repoData.Data); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	log.Info("Repository has been updated")

	return nil
}
