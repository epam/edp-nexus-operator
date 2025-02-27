package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestCreateBlobStore_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		blobStore               *nexusApi.NexusBlobStore
		nexusBlobStoreApiClient func(t *testing.T) nexus.FileBlobStore
		wantErr                 require.ErrorAssertionFunc
	}{
		{
			name: "create file blobstore successfully",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					SoftQuota: &nexusApi.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					File: &nexusApi.File{
						Path: "/test",
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.FileBlobStore {
				m := mocks.NewMockFileBlobStore(t)

				m.On("Get", "test-blobstore").
					Return(nil, errors.New("not found"))

				m.On("Create", &blobstore.File{
					Name: "test-blobstore",
					SoftQuota: &blobstore.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					Path: "/test",
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewCreateBlobStore(mocks.NewMockS3BlobStore(t), tt.nexusBlobStoreApiClient(t), fake.NewClientBuilder().Build())

			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.blobStore)
			tt.wantErr(t, err)
		})
	}
}
