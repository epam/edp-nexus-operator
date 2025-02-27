package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type RemoveBlobstore struct {
	nexusBlobStoreApiClient nexus.FileBlobStore
}

func NewRemoveBlobstore(nexusBlobStoreApiClient nexus.FileBlobStore) *RemoveBlobstore {
	return &RemoveBlobstore{nexusBlobStoreApiClient: nexusBlobStoreApiClient}
}

func (c *RemoveBlobstore) ServeRequest(ctx context.Context, blobStore *nexusApi.NexusBlobStore) error {
	log := ctrl.LoggerFrom(ctx).WithValues("blobstore_name", blobStore.Spec.Name)
	log.Info("Start removing blobstore")

	if err := c.nexusBlobStoreApiClient.Delete(blobStore.Spec.Name); err != nil {
		if !nexus.IsErrNotFound(err) {
			return fmt.Errorf("failed to delete blobstore: %w", err)
		}

		log.Info("Blobstore doesn't exist, skipping removal")
	}

	log.Info("Blobstore has been removed")

	return nil
}
