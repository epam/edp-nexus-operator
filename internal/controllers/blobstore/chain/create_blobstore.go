package chain

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type CreateBlobStore struct {
	nexusS3BlobStoreApiClient   nexus.S3BlobStore
	nexusFileBlobStoreApiClient nexus.FileBlobStore
	k8sClient                   client.Client
}

func NewCreateBlobStore(
	nexusS3BlobStoreApiClient nexus.S3BlobStore,
	nexusFileBlobStoreApiClient nexus.FileBlobStore,
	k8sClient client.Client,
) *CreateBlobStore {
	return &CreateBlobStore{nexusS3BlobStoreApiClient: nexusS3BlobStoreApiClient, nexusFileBlobStoreApiClient: nexusFileBlobStoreApiClient, k8sClient: k8sClient}
}

func (c *CreateBlobStore) ServeRequest(ctx context.Context, blobStore *nexusApi.NexusBlobStore) error {
	if blobStore.Spec.File != nil {
		return NewCreateFileBlobStore(c.nexusFileBlobStoreApiClient).ServeRequest(ctx, blobStore)
	}

	return NewCreateS3BlobStore(c.nexusS3BlobStoreApiClient, c.k8sClient).ServeRequest(ctx, blobStore)
}
