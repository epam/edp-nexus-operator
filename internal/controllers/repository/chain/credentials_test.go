package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

const (
	testNamespace = "default"
	leakSentinel  = "value-must-not-leak"
)

func secretRef(key string) *common.SecretKeySelector {
	return &common.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: "proxy-auth"},
		Key:                  key,
	}
}

func authSecret(data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "proxy-auth", Namespace: testNamespace},
		Data:       data,
	}
}

func credentialsSecret() *corev1.Secret {
	return authSecret(map[string][]byte{"user": []byte("secret-user"), "pass": []byte("secret-pass")})
}

func dockerProxySpec(auth *nexusApi.HTTPClientAuthentication) *nexusApi.NexusRepositorySpec {
	return &nexusApi.NexusRepositorySpec{
		Docker: &nexusApi.DockerSpec{
			Proxy: &nexusApi.DockerProxyRepository{
				ProxySpec: nexusApi.ProxySpec{
					Name:       "docker-proxy",
					HTTPClient: nexusApi.HTTPClient{Authentication: auth},
				},
			},
		},
	}
}

func mavenProxySpec(auth *nexusApi.HTTPClientAuthenticationWithPreemptive) *nexusApi.NexusRepositorySpec {
	return &nexusApi.NexusRepositorySpec{
		Maven: &nexusApi.MavenSpec{
			Proxy: &nexusApi.MavenProxyRepository{
				Name:       "maven-proxy",
				HTTPClient: nexusApi.HTTPClientWithPreemptiveAuth{Authentication: auth},
			},
		},
	}
}

func inlineAuth(username, password string) *nexusApi.HTTPClientAuthentication {
	return &nexusApi.HTTPClientAuthentication{Type: "username", Username: username, Password: password}
}

func TestCreateRepository_ServeRequest_SecretCredentials(t *testing.T) {
	t.Parallel()

	repository := &nexusApi.NexusRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "docker-proxy", Namespace: testNamespace},
		Spec: *dockerProxySpec(&nexusApi.HTTPClientAuthentication{
			Type:                 "username",
			UsernameSecretKeyRef: secretRef("user"),
			PasswordSecretKeyRef: secretRef("pass"),
		}),
	}
	original := repository.DeepCopy()

	m := mocks.NewMockRepository(t)
	m.On("Get", testifymock.Anything, "docker-proxy", nexus.FormatDocker, nexus.TypeProxy).Return(nil, nil)
	m.On("Update", testifymock.Anything, "docker-proxy", nexus.FormatDocker, nexus.TypeProxy,
		dockerProxySpec(inlineAuth("secret-user", "secret-pass")).Docker.Proxy).Return(nil)

	k8sClient := fake.NewClientBuilder().WithObjects(credentialsSecret()).Build()

	err := NewCreateRepository(m, k8sClient).
		ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), repository)
	require.NoError(t, err)
	require.Equal(t, original, repository, "the reconciled object must keep the references and get no credentials")
}

func TestGetRepoDataWithCredentials(t *testing.T) {
	t.Parallel()

	bothRefs := &nexusApi.HTTPClientAuthentication{
		Type: "username", UsernameSecretKeyRef: secretRef("user"), PasswordSecretKeyRef: secretRef("pass"),
	}
	passwordRef := &nexusApi.HTTPClientAuthentication{
		Type: "username", Username: "inline-user", PasswordSecretKeyRef: secretRef("pass"),
	}

	tests := []struct {
		name    string
		spec    *nexusApi.NexusRepositorySpec
		objects []client.Object
		want    *nexusApi.NexusRepositorySpec
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "inline credentials need no secret",
			spec:    dockerProxySpec(inlineAuth("inline-user", "inline-pass")),
			want:    dockerProxySpec(inlineAuth("inline-user", "inline-pass")),
			wantErr: require.NoError,
		},
		{
			name:    "no authentication needs no secret",
			spec:    dockerProxySpec(nil),
			want:    dockerProxySpec(nil),
			wantErr: require.NoError,
		},
		{
			name:    "credentials from secret",
			spec:    dockerProxySpec(bothRefs),
			objects: []client.Object{credentialsSecret()},
			want:    dockerProxySpec(inlineAuth("secret-user", "secret-pass")),
			wantErr: require.NoError,
		},
		{
			name:    "inline username and password from secret",
			spec:    dockerProxySpec(passwordRef),
			objects: []client.Object{credentialsSecret()},
			want:    dockerProxySpec(inlineAuth("inline-user", "secret-pass")),
			wantErr: require.NoError,
		},
		{
			name: "maven preemptive credentials from secret",
			spec: mavenProxySpec(&nexusApi.HTTPClientAuthenticationWithPreemptive{
				Type: "username", UsernameSecretKeyRef: secretRef("user"), PasswordSecretKeyRef: secretRef("pass"),
			}),
			objects: []client.Object{credentialsSecret()},
			want: mavenProxySpec(&nexusApi.HTTPClientAuthenticationWithPreemptive{
				Type: "username", Username: "secret-user", Password: "secret-pass",
			}),
			wantErr: require.NoError,
		},
		{
			name: "missing secret",
			spec: dockerProxySpec(passwordRef),
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), `secrets "proxy-auth" not found`)
			},
		},
		{
			name:    "missing key",
			spec:    dockerProxySpec(passwordRef),
			objects: []client.Object{authSecret(map[string][]byte{"other": []byte(leakSentinel)})},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), `key "pass" in secret "proxy-auth" is missing or empty`)
				require.NotContains(t, err.Error(), leakSentinel)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithObjects(tt.objects...).Build()
			original := tt.spec.DeepCopy()

			repoData, err := getRepoDataWithCredentials(context.Background(), k8sClient, testNamespace, tt.spec)

			require.Equal(t, original, tt.spec, "the caller's spec must not be mutated")

			tt.wantErr(t, err)

			if err != nil {
				return
			}

			want, err := nexus.GetRepoData(tt.want)
			require.NoError(t, err)
			require.Equal(t, want.Data, repoData.Data)
		})
	}
}
