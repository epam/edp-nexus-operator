package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

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
			name: "blobstore doesn't exist, creating new one",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					SoftQuota: &nexusApi.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					File: nexusApi.File{
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
		{
			name: "blobstore exists, updating it",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					SoftQuota: &nexusApi.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					File: nexusApi.File{
						Path: "/test",
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.FileBlobStore {
				m := mocks.NewMockFileBlobStore(t)

				m.On("Get", "test-blobstore").
					Return(&blobstore.File{
						Name: "test-blobstore",
						Path: "/test",
					}, nil)

				m.On("Update", "test-blobstore", &blobstore.File{
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
		{
			name: "blobstore exists, no changes",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					SoftQuota: &nexusApi.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceUsedQuota,
						Limit: 100,
					},
					File: nexusApi.File{
						Path: "/test",
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.FileBlobStore {
				m := mocks.NewMockFileBlobStore(t)

				m.On("Get", "test-blobstore").
					Return(&blobstore.File{
						Name: "test-blobstore",
						SoftQuota: &blobstore.SoftQuota{
							Type:  nexusApi.SoftQuotaSpaceUsedQuota,
							Limit: 100,
						},
						Path: "/test",
					}, nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to update blobstore",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					SoftQuota: &nexusApi.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					File: nexusApi.File{
						Path: "/test",
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.FileBlobStore {
				m := mocks.NewMockFileBlobStore(t)

				m.On("Get", "test-blobstore").
					Return(&blobstore.File{
						Name: "test-blobstore",
						Path: "/test",
					}, nil)

				m.On("Update", "test-blobstore", &blobstore.File{
					Name: "test-blobstore",
					SoftQuota: &blobstore.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					Path: "/test",
				}).Return(errors.New("failed to update blobstore"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update blobstore")
			},
		},
		{
			name: "failed to create blobstore",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					SoftQuota: &nexusApi.SoftQuota{
						Type:  nexusApi.SoftQuotaSpaceRemainingQuota,
						Limit: 100,
					},
					File: nexusApi.File{
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
				}).Return(errors.New("failed to create blobstore"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create blobstore")
			},
		},
		{
			name: "failed to get blobstore",
			blobStore: &nexusApi.NexusBlobStore{
				Spec: nexusApi.NexusBlobStoreSpec{
					Name: "test-blobstore",
					File: nexusApi.File{
						Path: "/test",
					},
				},
			},
			nexusBlobStoreApiClient: func(t *testing.T) nexus.FileBlobStore {
				m := mocks.NewMockFileBlobStore(t)

				m.On("Get", "test-blobstore").
					Return(nil, errors.New("failed to get blobstore"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get blobstore")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewCreateBlobStore(tt.nexusBlobStoreApiClient(t))

			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.blobStore)
			tt.wantErr(t, err)
		})
	}
}
