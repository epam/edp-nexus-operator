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

func TestCreateRepository_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                     string
		repository               *nexusApi.NexusRepository
		nexusRepositoryApiClient func(t *testing.T) nexus.Repository
		wantErr                  require.ErrorAssertionFunc
	}{
		{
			name: "repository doesn't exist, creating new one",
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
				m := mocks.NewMockRepository(t)

				m.On("Get", testifymock.Anything, "go-proxy", nexus.FormatGo, nexus.TypeProxy).
					Return(nil, nexus.ErrNotFound)
				m.On("Create", testifymock.Anything, nexus.FormatGo, nexus.TypeProxy, &nexusApi.GoProxyRepository{
					ProxySpec: nexusApi.ProxySpec{
						Name: "go-proxy",
					},
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "repository exist, updating it",
			repository: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name:   "go-proxy",
								Online: true,
							},
						},
					},
				},
			},
			nexusRepositoryApiClient: func(t *testing.T) nexus.Repository {
				m := mocks.NewMockRepository(t)

				m.On("Get", testifymock.Anything, "go-proxy", nexus.FormatGo, nexus.TypeProxy).
					Return(nil, nil)
				m.On("Update", testifymock.Anything, "go-proxy", nexus.FormatGo, nexus.TypeProxy, &nexusApi.GoProxyRepository{
					ProxySpec: nexusApi.ProxySpec{
						Name:   "go-proxy",
						Online: true,
					},
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to update repository",
			repository: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name:   "go-proxy",
								Online: true,
							},
						},
					},
				},
			},
			nexusRepositoryApiClient: func(t *testing.T) nexus.Repository {
				m := mocks.NewMockRepository(t)

				m.On("Get", testifymock.Anything, "go-proxy", nexus.FormatGo, nexus.TypeProxy).
					Return(nil, nil)
				m.On("Update", testifymock.Anything, "go-proxy", nexus.FormatGo, nexus.TypeProxy, &nexusApi.GoProxyRepository{
					ProxySpec: nexusApi.ProxySpec{
						Name:   "go-proxy",
						Online: true,
					},
				}).Return(errors.New("failed to update repository"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update repository")
			},
		},
		{
			name: "failed to create repository",
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
				m := mocks.NewMockRepository(t)

				m.On("Get", testifymock.Anything, "go-proxy", nexus.FormatGo, nexus.TypeProxy).
					Return(nil, nexus.ErrNotFound)
				m.On("Create", testifymock.Anything, nexus.FormatGo, nexus.TypeProxy, &nexusApi.GoProxyRepository{
					ProxySpec: nexusApi.ProxySpec{
						Name: "go-proxy",
					},
				}).Return(errors.New("failed to create repository"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create repository")
			},
		},
		{
			name: "failed to get repository",
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
				m := mocks.NewMockRepository(t)

				m.On("Get", testifymock.Anything, "go-proxy", nexus.FormatGo, nexus.TypeProxy).
					Return(nil, errors.New("failed to get repository"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get repository")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewCreateRepository(tt.nexusRepositoryApiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.repository)
			tt.wantErr(t, err)
		})
	}
}
