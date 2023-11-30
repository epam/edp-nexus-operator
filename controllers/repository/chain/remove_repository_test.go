package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestRemoveRepository_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                     string
		repository               *nexusApi.NexusRepository
		nexusRepositoryApiClient func(t *testing.T) nexus.Repository
		wantErr                  require.ErrorAssertionFunc
	}{
		{
			name: "successfully removed",
			repository: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
				},
			},
			nexusRepositoryApiClient: func(t *testing.T) nexus.Repository {
				m := mocks.NewRepository(t)

				m.On("Delete", testifymock.Anything, "go-proxy").Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "repository doesn't exist, skipping removal",
			repository: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
				},
			},
			nexusRepositoryApiClient: func(t *testing.T) nexus.Repository {
				m := mocks.NewRepository(t)

				m.On("Delete", testifymock.Anything, "go-proxy").Return(nexus.ErrNotFound)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to remove repository",
			repository: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
				},
			},
			nexusRepositoryApiClient: func(t *testing.T) nexus.Repository {
				m := mocks.NewRepository(t)

				m.On("Delete", testifymock.Anything, "go-proxy").
					Return(errors.New("failed to remove repository"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete repository")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewRemoveRepository(tt.nexusRepositoryApiClient(t))

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.repository)
			tt.wantErr(t, err)
		})
	}
}
