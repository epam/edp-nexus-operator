package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestRemoveBlobstore_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		blobStore               *nexusApi.NexusBlobStore
		nexusBlobStoreApiClient func(t *testing.T) nexus.FileBlobStore
		wantErr                 require.ErrorAssertionFunc
	}{
		{
			name: "removing blobstore successfully",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.FileBlobStore {
				m := mocks.NewMockFileBlobStore(t)

				m.On("Delete", "test-blobstore").Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "blobstore doesn't exist, skipping removal",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.FileBlobStore {
				m := mocks.NewMockFileBlobStore(t)

				m.On("Delete", "test-blobstore").Return(nexus.ErrNotFound)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "blobstore removal failed",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.FileBlobStore {
				m := mocks.NewMockFileBlobStore(t)

				m.On("Delete", "test-blobstore").Return(errors.New("failed to remove blobstore"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to remove blobstore")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewRemoveBlobstore(tt.nexusBlobStoreApiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.blobStore)
			tt.wantErr(t, err)
		})
	}
}
