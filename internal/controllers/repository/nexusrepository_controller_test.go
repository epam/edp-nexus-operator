package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

func goProxyRepository(name, ns string, auth *nexusApi.HTTPClientAuthentication) *nexusApi.NexusRepository {
	return &nexusApi.NexusRepository{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: nexusApi.NexusRepositorySpec{
			NexusRef: common.NexusRef{Name: nexusCRName},
			Go: &nexusApi.GoSpec{
				Proxy: &nexusApi.GoProxyRepository{
					ProxySpec: nexusApi.ProxySpec{
						Name:          name,
						Online:        true,
						Storage:       nexusApi.Storage{BlobStoreName: "default", StrictContentTypeValidation: true},
						Proxy:         nexusApi.Proxy{ContentMaxAge: 1440, MetadataMaxAge: 1440, RemoteURL: "https://go-proxy-url"},
						NegativeCache: nexusApi.NegativeCache{Enabled: true, TTL: 1440},
						HTTPClient:    nexusApi.HTTPClient{AutoBlock: true, Authentication: auth},
					},
				},
			},
		},
	}
}

func selector(name, key string) *common.SecretKeySelector {
	return &common.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: name},
		Key:                  key,
	}
}

func TestNexusRepositoryReconciler_mapSecretToNexusRepository(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, nexusApi.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		goProxyRepository("by-password", "ns", &nexusApi.HTTPClientAuthentication{
			Username:             "user",
			PasswordSecretKeyRef: selector("proxy-auth", "pass"),
		}),
		goProxyRepository("by-username", "ns", &nexusApi.HTTPClientAuthentication{
			UsernameSecretKeyRef: selector("proxy-auth", "user"),
			Password:             "pass",
		}),
		goProxyRepository("other-secret", "ns", &nexusApi.HTTPClientAuthentication{
			UsernameSecretKeyRef: selector("another", "user"),
			PasswordSecretKeyRef: selector("another", "pass"),
		}),
		goProxyRepository("inline", "ns", &nexusApi.HTTPClientAuthentication{Username: "user", Password: "pass"}),
		goProxyRepository("anonymous", "ns", nil),
		goProxyRepository("other-namespace", "other", &nexusApi.HTTPClientAuthentication{
			UsernameSecretKeyRef: selector("proxy-auth", "user"),
			PasswordSecretKeyRef: selector("proxy-auth", "pass"),
		}),
	).Build()

	got := NewNexusRepositoryReconciler(k8sClient, nil).mapSecretToNexusRepository(
		context.Background(),
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "proxy-auth", Namespace: "ns"}},
	)

	names := make([]string, 0, len(got))

	for _, req := range got {
		require.Equal(t, "ns", req.Namespace)

		names = append(names, req.Name)
	}

	require.ElementsMatch(t, []string{"by-password", "by-username"}, names)
}
