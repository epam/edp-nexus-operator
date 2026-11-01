package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

func repositoryWithRef(name, namespace, secretName string) *nexusApi.NexusRepository {
	selector := &common.SecretKeySelector{Key: "password"}
	selector.Name = secretName

	return &nexusApi.NexusRepository{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: nexusApi.NexusRepositorySpec{
			Docker: &nexusApi.DockerSpec{Proxy: &nexusApi.DockerProxyRepository{
				ProxySpec: nexusApi.ProxySpec{
					Name: name,
					HTTPClient: nexusApi.HTTPClient{Authentication: &nexusApi.HTTPClientAuthentication{
						Type: "username", Username: "alice", PasswordRef: selector,
					}},
				},
			}},
		},
	}
}

func TestIndexPasswordRef(t *testing.T) {
	t.Parallel()

	maven := repositoryWithRef("maven", "ns", "unused")
	maven.Spec = nexusApi.NexusRepositorySpec{Maven: &nexusApi.MavenSpec{Proxy: &nexusApi.MavenProxyRepository{
		Name: "maven",
		HTTPClient: nexusApi.HTTPClientWithPreemptiveAuth{Authentication: &nexusApi.HTTPClientAuthenticationWithPreemptive{
			Username: "alice", PasswordRef: repositoryWithRef("x", "ns", "maven-cred").Spec.Docker.Proxy.Authentication.PasswordRef,
		}},
	}}}

	plain := repositoryWithRef("plain", "ns", "unused")
	plain.Spec.Docker.Proxy.Authentication.PasswordRef = nil
	plain.Spec.Docker.Proxy.Authentication.Password = "pw" //nolint:staticcheck // covers the deprecated password field

	noAuth := repositoryWithRef("noauth", "ns", "unused")
	noAuth.Spec.Docker.Proxy.Authentication = nil

	hosted := &nexusApi.NexusRepository{Spec: nexusApi.NexusRepositorySpec{
		Docker: &nexusApi.DockerSpec{Hosted: &nexusApi.DockerHostedRepository{}},
	}}

	empty := &nexusApi.NexusRepository{}

	require.Equal(t, []string{"docker-cred"}, indexPasswordRef(repositoryWithRef("docker", "ns", "docker-cred")))
	require.Equal(t, []string{"maven-cred"}, indexPasswordRef(maven))
	require.Nil(t, indexPasswordRef(plain))
	require.Nil(t, indexPasswordRef(noAuth))
	require.Nil(t, indexPasswordRef(hosted))
	require.Nil(t, indexPasswordRef(empty))
	require.Nil(t, indexPasswordRef(&corev1.Secret{}))
}

func TestMapSecretToRepositories(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, nexusApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&nexusApi.NexusRepository{}, passwordRefIndex, indexPasswordRef).
		WithObjects(
			repositoryWithRef("a", "ns", "cred"),
			repositoryWithRef("b", "ns", "cred"),
			repositoryWithRef("other-secret", "ns", "other"),
			repositoryWithRef("other-ns", "ns2", "cred"),
		).
		Build()

	r := NewNexusRepositoryReconciler(k8sClient, nil)

	got := r.mapSecretToRepositories(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "cred", Namespace: "ns"},
	})
	require.ElementsMatch(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "a"}},
		{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "b"}},
	}, got)

	require.Empty(t, r.mapSecretToRepositories(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "ns"},
	}))
}

func TestSecretDataChangedPredicate(t *testing.T) {
	t.Parallel()

	p := secretDataChangedPredicate()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "cred", Namespace: "ns", Labels: map[string]string{"a": "1"}},
		Data:       map[string][]byte{"password": []byte("pw-1")},
	}

	relabeled := secret.DeepCopy()
	relabeled.Labels["a"] = "2"

	rotated := secret.DeepCopy()
	rotated.Data["password"] = []byte("pw-2")

	var other client.Object = &nexusApi.NexusRepository{}

	require.False(t, p.Update(event.UpdateEvent{ObjectOld: secret, ObjectNew: relabeled}), "label change")
	require.True(t, p.Update(event.UpdateEvent{ObjectOld: secret, ObjectNew: rotated}), "data change")
	require.False(t, p.Update(event.UpdateEvent{ObjectOld: other, ObjectNew: other}), "not a Secret")
	require.True(t, p.Create(event.CreateEvent{Object: secret}))
	require.True(t, p.Delete(event.DeleteEvent{Object: secret}))
	require.True(t, p.Generic(event.GenericEvent{Object: secret}))
}
