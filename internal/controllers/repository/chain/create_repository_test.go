package chain

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewCreateRepository(tt.nexusRepositoryApiClient(t), fake.NewClientBuilder().Build())
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.repository)
			tt.wantErr(t, err)
		})
	}
}

const hugeInt = 9007199254740993 // 2^53 + 1, not representable as float64

func dockerProxyWithAuth(auth *nexusApi.HTTPClientAuthentication) *nexusApi.NexusRepository {
	return &nexusApi.NexusRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "docker-proxy", Namespace: testNamespace},
		Spec: nexusApi.NexusRepositorySpec{
			Docker: &nexusApi.DockerSpec{
				Proxy: &nexusApi.DockerProxyRepository{
					ProxySpec: nexusApi.ProxySpec{
						Name:   "docker-proxy",
						Online: true,
						Proxy: nexusApi.Proxy{
							ContentMaxAge: hugeInt,
							RemoteURL:     "https://remote.example",
						},
						HTTPClient: nexusApi.HTTPClient{AutoBlock: true, Authentication: auth},
					},
				},
			},
		},
	}
}

// captureRepositoryClient answers Get with "exists" and records the JSON of every Update body.
func captureRepositoryClient(t *testing.T, bodies *[]string, updateErr error) nexus.Repository {
	t.Helper()

	m := mocks.NewMockRepository(t)
	m.On("Get", testifymock.Anything, "docker-proxy", nexus.FormatDocker, nexus.TypeProxy).Return(nil, nil)
	m.On("Update", testifymock.Anything, "docker-proxy", nexus.FormatDocker, nexus.TypeProxy, testifymock.Anything).
		Run(func(args testifymock.Arguments) {
			raw, err := json.Marshal(args.Get(4))
			require.NoError(t, err)

			*bodies = append(*bodies, string(raw))
		}).
		Return(updateErr)

	return m
}

func serve(t *testing.T, c *CreateRepository, repository *nexusApi.NexusRepository) error {
	t.Helper()

	return c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), repository)
}

func TestCreateRepository_ServeRequest_passwordRef(t *testing.T) {
	t.Parallel()

	repository := dockerProxyWithAuth(&nexusApi.HTTPClientAuthentication{
		Type: "username", Username: "alice", PasswordRef: ref("cred", "password"),
	})

	var bodies []string

	c := NewCreateRepository(
		captureRepositoryClient(t, &bodies, nil),
		newFakeClient(t, map[string][]byte{"password": []byte("pw-1")}),
	)
	require.NoError(t, serve(t, c, repository))
	require.Len(t, bodies, 1)

	require.JSONEq(t, `{
		"name": "docker-proxy",
		"online": true,
		"storage": {"blobStoreName": "", "strictContentTypeValidation": false},
		"proxy": {"contentMaxAge": 9007199254740993, "metadataMaxAge": 0, "remoteUrl": "https://remote.example"},
		"negativeCache": {"enabled": false, "timeToLive": 0},
		"httpClient": {
			"autoBlock": true,
			"blocked": false,
			"authentication": {"type": "username", "username": "alice", "password": "pw-1"}
		},
		"docker": {"forceBasicAuth": false, "v1Enabled": false},
		"dockerProxy": {"indexType": ""}
	}`, bodies[0])
	require.NotContains(t, bodies[0], "passwordRef")

	auth := repository.Spec.Docker.Proxy.Authentication
	require.Empty(t, auth.Password, "the resolved password is not written to the resource") //nolint:staticcheck // covers the deprecated password field
	require.Equal(t, ref("cred", "password"), auth.PasswordRef)
}

func TestCreateRepository_ServeRequest_plainPasswordBodyUnchanged(t *testing.T) {
	t.Parallel()

	repository := dockerProxyWithAuth(&nexusApi.HTTPClientAuthentication{
		Type: "username", Username: "alice", Password: "pw-plain", //nolint:staticcheck // covers the deprecated password field
	})

	want, err := json.Marshal(repository.Spec.Docker.Proxy)
	require.NoError(t, err)

	var bodies []string

	c := NewCreateRepository(captureRepositoryClient(t, &bodies, nil), newFakeClient(t, nil))
	require.NoError(t, serve(t, c, repository))
	require.Len(t, bodies, 1)
	require.JSONEq(t, string(want), bodies[0])
	require.Contains(t, bodies[0], "9007199254740993")
}

func TestCreateRepository_ServeRequest_withoutAuthenticationBodyUnchanged(t *testing.T) {
	t.Parallel()

	repository := dockerProxyWithAuth(nil)

	want, err := json.Marshal(repository.Spec.Docker.Proxy)
	require.NoError(t, err)

	var bodies []string

	c := NewCreateRepository(captureRepositoryClient(t, &bodies, nil), newFakeClient(t, nil))
	require.NoError(t, serve(t, c, repository))
	require.Equal(t, []string{string(want)}, bodies)
}

func TestCreateRepository_ServeRequest_hostedAndGroupBodyUnchanged(t *testing.T) {
	t.Parallel()

	hosted := &nexusApi.HostedSpec{Name: "docker-hosted", Online: true}
	group := &nexusApi.DockerGroupRepository{Name: "docker-group", Online: true}

	tests := []struct {
		name       string
		repository *nexusApi.NexusRepository
		repoName   string
		repoType   string
		data       any
	}{
		{
			name: "hosted",
			repository: &nexusApi.NexusRepository{Spec: nexusApi.NexusRepositorySpec{
				Docker: &nexusApi.DockerSpec{Hosted: &nexusApi.DockerHostedRepository{HostedSpec: *hosted}},
			}},
			repoName: "docker-hosted",
			repoType: nexus.TypeHosted,
			data:     &nexusApi.DockerHostedRepository{HostedSpec: *hosted},
		},
		{
			name: "group",
			repository: &nexusApi.NexusRepository{Spec: nexusApi.NexusRepositorySpec{
				Docker: &nexusApi.DockerSpec{Group: group},
			}},
			repoName: "docker-group",
			repoType: nexus.TypeGroup,
			data:     group,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := mocks.NewMockRepository(t)
			m.On("Get", testifymock.Anything, tt.repoName, nexus.FormatDocker, tt.repoType).Return(nil, nil)
			m.On("Update", testifymock.Anything, tt.repoName, nexus.FormatDocker, tt.repoType, tt.data).Return(nil)

			require.NoError(t, serve(t, NewCreateRepository(m, newFakeClient(t, nil)), tt.repository))
		})
	}
}

func TestCreateRepository_ServeRequest_rotation(t *testing.T) {
	t.Parallel()

	repository := dockerProxyWithAuth(&nexusApi.HTTPClientAuthentication{
		Type: "username", Username: "alice", PasswordRef: ref("cred", "password"),
	})
	k8sClient := newFakeClient(t, map[string][]byte{"password": []byte("pw-1")})

	var bodies []string

	c := NewCreateRepository(captureRepositoryClient(t, &bodies, nil), k8sClient)
	require.NoError(t, serve(t, c, repository))

	secret := &corev1.Secret{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: "cred"}, secret))
	secret.Data["password"] = []byte("pw-2")
	require.NoError(t, k8sClient.Update(context.Background(), secret))

	require.NoError(t, serve(t, c, repository))
	require.Len(t, bodies, 2)
	require.Contains(t, bodies[0], `"password":"pw-1"`)
	require.Contains(t, bodies[1], `"password":"pw-2"`)
}

func TestCreateRepository_ServeRequest_redactsResolvedPassword(t *testing.T) {
	t.Parallel()

	repository := dockerProxyWithAuth(&nexusApi.HTTPClientAuthentication{
		Type: "username", Username: "alice", PasswordRef: ref("cred", "password"),
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		var bodies []string

		c := NewCreateRepository(
			captureRepositoryClient(t, &bodies, errors.New("failed to update repository: password pw-1 is invalid for alice")),
			newFakeClient(t, map[string][]byte{"password": []byte("pw-1")}),
		)

		err := serve(t, c, repository)
		require.EqualError(t, err, "failed to update repository: failed to update repository: password <redacted> is invalid for alice")
	})

	t.Run("create", func(t *testing.T) {
		t.Parallel()

		m := mocks.NewMockRepository(t)
		m.On("Get", testifymock.Anything, "docker-proxy", nexus.FormatDocker, nexus.TypeProxy).Return(nil, nexus.ErrNotFound)
		m.On("Create", testifymock.Anything, nexus.FormatDocker, nexus.TypeProxy, testifymock.Anything).
			Return(errors.New("failed to create repository: bad pw-1"))

		err := serve(t, NewCreateRepository(m, newFakeClient(t, map[string][]byte{"password": []byte("pw-1")})), repository)
		require.EqualError(t, err, "failed to create repository: failed to create repository: bad <redacted>")
	})

	t.Run("plain password is not redacted", func(t *testing.T) {
		t.Parallel()

		plain := dockerProxyWithAuth(&nexusApi.HTTPClientAuthentication{
			Type: "username", Username: "alice", Password: "pw-plain", //nolint:staticcheck // covers the deprecated password field
		})

		var bodies []string

		c := NewCreateRepository(
			captureRepositoryClient(t, &bodies, errors.New("rejected pw-plain")),
			newFakeClient(t, nil),
		)

		require.ErrorContains(t, serve(t, c, plain), "rejected pw-plain")
	})
}

func TestCreateRepository_ServeRequest_authenticationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		auth    *nexusApi.HTTPClientAuthentication
		data    map[string][]byte
		wantErr string
	}{
		{
			name:    "both password sources",
			auth:    &nexusApi.HTTPClientAuthentication{Type: "username", Username: "a", Password: "p", PasswordRef: ref("cred", "password")}, //nolint:staticcheck // covers the deprecated password field
			wantErr: "mutually exclusive",
		},
		{
			name:    "no password source",
			auth:    &nexusApi.HTTPClientAuthentication{Type: "username", Username: "a"},
			wantErr: "one of password or passwordRef must be set",
		},
		{
			name:    "missing secret",
			auth:    &nexusApi.HTTPClientAuthentication{Type: "username", Username: "a", PasswordRef: ref("absent", "password")},
			wantErr: "unable to get secret absent",
		},
		{
			name:    "empty value",
			auth:    &nexusApi.HTTPClientAuthentication{Type: "username", Username: "a", PasswordRef: ref("cred", "password")},
			data:    map[string][]byte{"password": {}},
			wantErr: "secret cred key password is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// No Nexus call is expected: the mock fails the test on any.
			c := NewCreateRepository(mocks.NewMockRepository(t), newFakeClient(t, tt.data))
			require.ErrorContains(t, serve(t, c, dockerProxyWithAuth(tt.auth)), tt.wantErr)
		})
	}
}

func TestCreateRepository_requestBody_proxyWithoutSettings(t *testing.T) {
	t.Parallel()

	c := NewCreateRepository(mocks.NewMockRepository(t), newFakeClient(t, nil))

	_, _, err := c.requestBody(context.Background(), testNamespace, &nexus.RepoData{
		Type: nexus.TypeProxy, Format: nexus.FormatGo, Name: "go-proxy",
	})
	require.ErrorContains(t, err, "proxy repository go-proxy has no proxy settings")
}

func TestCreateRepository_ServeRequest_mavenPreemptiveOnly(t *testing.T) {
	t.Parallel()

	preemptive := true
	repository := &nexusApi.NexusRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "maven-proxy", Namespace: testNamespace},
		Spec: nexusApi.NexusRepositorySpec{Maven: &nexusApi.MavenSpec{Proxy: &nexusApi.MavenProxyRepository{
			Name: "maven-proxy",
			HTTPClient: nexusApi.HTTPClientWithPreemptiveAuth{
				Authentication: &nexusApi.HTTPClientAuthenticationWithPreemptive{Type: "username", Preemptive: &preemptive},
			},
		}}},
	}

	var body string

	m := mocks.NewMockRepository(t)
	m.On("Get", testifymock.Anything, "maven-proxy", nexus.FormatMaven, nexus.TypeProxy).Return(nil, nil)
	m.On("Update", testifymock.Anything, "maven-proxy", nexus.FormatMaven, nexus.TypeProxy, testifymock.Anything).
		Run(func(args testifymock.Arguments) {
			raw, err := json.Marshal(args.Get(4))
			require.NoError(t, err)

			body = string(raw)
		}).
		Return(nil)

	require.NoError(t, serve(t, NewCreateRepository(m, newFakeClient(t, nil)), repository))
	require.Contains(t, body, `"authentication":{"type":"username","preemptive":true}`)
}
