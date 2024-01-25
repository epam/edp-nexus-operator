package chain

import (
	"context"
	"fmt"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type CreateBlobStore struct {
	nexusBlobStoreApiClient nexus.FileBlobStore
}

func NewCreateBlobStore(nexusBlobStoreApiClient nexus.FileBlobStore) *CreateBlobStore {
	return &CreateBlobStore{nexusBlobStoreApiClient: nexusBlobStoreApiClient}
}

func (c *CreateBlobStore) ServeRequest(ctx context.Context, blobStore *nexusApi.NexusBlobStore) error {
	log := ctrl.LoggerFrom(ctx).WithValues("blobstore_name", blobStore.Spec.Name)
	log.Info("Start creating blobstore")

	nexusBlobStore, err := c.nexusBlobStoreApiClient.Get(blobStore.Spec.Name)
	if err != nil {
		if !nexus.IsErrNotFound(err) {
			return fmt.Errorf("failed to get blobstore: %w", err)
		}

		log.Info("Blobstore doesn't exist, creating new one")

		if err = c.nexusBlobStoreApiClient.Create(specToBlobstore(&blobStore.Spec)); err != nil {
			return fmt.Errorf("failed to create blobstore: %w", err)
		}

		log.Info("Blobstore has been created")

		return nil
	}

	newNexusBlobStore := specToBlobstore(&blobStore.Spec)

	if blobstoreChanged(newNexusBlobStore, nexusBlobStore) {
		log.Info("Updating blobstore")

		if err = c.nexusBlobStoreApiClient.Update(blobStore.Spec.Name, newNexusBlobStore); err != nil {
			return fmt.Errorf("failed to update blobstore: %w", err)
		}

		log.Info("Blobstore has been updated")
	}

	return nil
}

func specToBlobstore(spec *nexusApi.NexusBlobStoreSpec) *blobstore.File {
	f := &blobstore.File{
		Name: spec.Name,
		Path: spec.File.Path,
	}

	if spec.SoftQuota != nil {
		f.SoftQuota = &blobstore.SoftQuota{
			Limit: spec.SoftQuota.Limit,
			Type:  spec.SoftQuota.Type,
		}
	}

	return f
}

func blobstoreChanged(newNexusBlobStore, nexusBlobStore *blobstore.File) bool {
	if newNexusBlobStore.Path != nexusBlobStore.Path {
		return true
	}

	if newNexusBlobStore.SoftQuota != nil && nexusBlobStore.SoftQuota != nil {
		return newNexusBlobStore.SoftQuota.Limit != nexusBlobStore.SoftQuota.Limit ||
			newNexusBlobStore.SoftQuota.Type != nexusBlobStore.SoftQuota.Type
	}

	return (newNexusBlobStore.SoftQuota != nil && nexusBlobStore.SoftQuota == nil) ||
		(newNexusBlobStore.SoftQuota == nil && nexusBlobStore.SoftQuota != nil)
}
