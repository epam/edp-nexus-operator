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

	repoData, err := nexus.GetRepoData(&repository.Spec)
	if err != nil {
		return fmt.Errorf("failed to get repository data: %w", err)
	}

	log = log.WithValues("type", repoData.Type, "format", repoData.Format, "name", repoData.Name)

	body, secretPassword, err := c.requestBody(ctx, repository.Namespace, repoData)
	if err != nil {
		return err
	}

	log.Info("Getting repository")

	_, err = c.nexusRepositoryApiClient.Get(ctx, repoData.Name, repoData.Format, repoData.Type)
	if err != nil {
		if errors.Is(err, nexus.ErrNotFound) {
			log.Info("Repository doesn't exist, creating new one")

			if err = c.nexusRepositoryApiClient.Create(ctx, repoData.Format, repoData.Type, body); err != nil {
				return fmt.Errorf("failed to create repository: %w", redactError(err, secretPassword))
			}

			log.Info("Repository has been created")

			return nil
		}

		return fmt.Errorf("failed to get repository: %w", err)
	}

	log.Info("Updating repository")

	if err = c.nexusRepositoryApiClient.Update(ctx, repoData.Name, repoData.Format, repoData.Type, body); err != nil {
		return fmt.Errorf("failed to update repository: %w", redactError(err, secretPassword))
	}

	log.Info("Repository has been updated")

	return nil
}

// requestBody returns the request body and the password read from a Secret.
// A proxy repository with authentication gets a body with the resolved password; any other body is the spec itself.
func (c *CreateRepository) requestBody(
	ctx context.Context,
	namespace string,
	repoData *nexus.RepoData,
) (body any, secretPassword string, err error) {
	if repoData.Type != nexus.TypeProxy {
		return repoData.Data, "", nil
	}

	if repoData.Proxy == nil {
		return nil, "", fmt.Errorf("proxy repository %s has no proxy settings", repoData.Name)
	}

	if repoData.Proxy.Auth == nil {
		return repoData.Data, "", nil
	}

	resolved, err := resolveAuthentication(
		ctx,
		c.k8sClient,
		namespace,
		repoData.Proxy.Auth,
		repoData.Format == nexus.FormatMaven,
	)
	if err != nil {
		return nil, "", err
	}

	body, err = nexus.WithAuthentication(repoData.Data, resolved.wire)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build repository request body: %w", err)
	}

	return body, resolved.secretPassword, nil
}
