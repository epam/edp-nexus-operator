package nexus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

func TestApiClientProvider_GetNexusApiClientFromNexus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nexus     *nexusApi.Nexus
		k8sClient func(t *testing.T) client.Client
		want      require.ValueAssertionFunc
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "successfully get nexus api client",
			nexus: &nexusApi.Nexus{
				Spec: nexusApi.NexusSpec{
					Secret: "nexus-secret",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name: "nexus-secret",
							},
							Data: map[string][]byte{
								"user":     []byte("user"),
								"password": []byte("password"),
							},
						},
					).
					Build()
			},
			want:    require.NotNil,
			wantErr: require.NoError,
		},
		{
			name: "failed to get nexus password",
			nexus: &nexusApi.Nexus{
				Spec: nexusApi.NexusSpec{
					Secret: "nexus-secret",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name: "nexus-secret",
							},
							Data: map[string][]byte{
								"user": []byte("user"),
							},
						},
					).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "nexus secret doesn't contain password")
			},
		},
		{
			name: "failed to get nexus user",
			nexus: &nexusApi.Nexus{
				Spec: nexusApi.NexusSpec{
					Secret: "nexus-secret",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name: "nexus-secret",
							},
						},
					).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "nexus secret doesn't contain user")
			},
		},
		{
			name: "failed to get nexus secret",
			nexus: &nexusApi.Nexus{
				Spec: nexusApi.NexusSpec{
					Secret: "nexus-secret",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get nexus secret")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewApiClientProvider(tt.k8sClient(t))
			got, err := p.GetNexusApiClientFromNexus(context.Background(), tt.nexus)

			tt.wantErr(t, err)
			tt.want(t, got)
		})
	}
}

func TestApiClientProvider_GetNexusApiClientFromNexusRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       common.HasNexusRef
		k8sClient func(t *testing.T) client.Client
		want      require.ValueAssertionFunc
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "successfully get nexus api client",
			ref: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "nexus-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"user":     []byte("user"),
								"password": []byte("password"),
							},
						},
						&nexusApi.Nexus{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "nexus",
								Namespace: "default",
							},
							Spec: nexusApi.NexusSpec{
								Secret: "nexus-secret",
							},
						},
					).
					Build()
			},
			want:    require.NotNil,
			wantErr: require.NoError,
		},
		{
			name: "failed to get nexus instance",
			ref: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get nexus instance")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewApiClientProvider(tt.k8sClient(t))
			got, err := p.GetNexusApiClientFromNexusRef(context.Background(), "default", tt.ref)

			tt.wantErr(t, err)
			tt.want(t, got)
		})
	}
}

func TestApiClientProvider_GetNexusRepositoryClientFromNexusRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       common.HasNexusRef
		k8sClient func(t *testing.T) client.Client
		want      require.ValueAssertionFunc
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "successfully get nexus api client",
			ref: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "nexus-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"user":     []byte("user"),
								"password": []byte("password"),
							},
						},
						&nexusApi.Nexus{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "nexus",
								Namespace: "default",
							},
							Spec: nexusApi.NexusSpec{
								Secret: "nexus-secret",
							},
						},
					).
					Build()
			},
			want:    require.NotNil,
			wantErr: require.NoError,
		},
		{
			name: "secret not found",
			ref: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					WithObjects(
						&nexusApi.Nexus{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "nexus",
								Namespace: "default",
							},
							Spec: nexusApi.NexusSpec{
								Secret: "nexus-secret",
							},
						},
					).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get nexus secret")
			},
		},
		{
			name: "failed to get nexus instance",
			ref: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				sh := runtime.NewScheme()
				require.NoError(t, nexusApi.AddToScheme(sh))
				require.NoError(t, corev1.AddToScheme(sh))

				return fake.NewClientBuilder().
					WithScheme(sh).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get nexus instance")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewApiClientProvider(tt.k8sClient(t))
			got, err := p.GetNexusRepositoryClientFromNexusRef(context.Background(), "default", tt.ref)

			tt.wantErr(t, err)
			tt.want(t, got)
		})
	}
}
