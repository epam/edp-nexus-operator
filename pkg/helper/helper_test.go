package helper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-nexus-operator/api/common"
)

func TestGetWatchNamespace(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T)
		want    string
	}{
		{
			name: "namespace is set",
			prepare: func(t *testing.T) {
				t.Setenv(WatchNamespaceEnvVar, "test")
			},
			want: "test",
		},
		{
			name:    "namespace is  not set",
			prepare: func(t *testing.T) {},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			got := GetWatchNamespace()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetDebugMode(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T)
		want    bool
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "debug mode is set",
			prepare: func(t *testing.T) {
				t.Setenv(debugModeEnvVar, "true")
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name:    "debug mode is  not set",
			prepare: func(t *testing.T) {},
			want:    false,
			wantErr: require.NoError,
		},
		{
			name: "debug mode is not a bool",
			prepare: func(t *testing.T) {
				t.Setenv(debugModeEnvVar, "not-a-bool")
			},
			want: false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid syntax")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			got, err := GetDebugMode()

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetValueFromSourceRef(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name      string
		sourceRef *common.SourceRef
		k8sClient func(t *testing.T) client.Client
		want      string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "get value from config map",
			sourceRef: &common.SourceRef{
				ConfigMapKeyRef: &common.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-configmap",
					},
					Key: "test-key",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "default",
						},
						Data: map[string]string{
							"test-key": "test-value",
						},
					},
				).Build()
			},
			want:    "test-value",
			wantErr: require.NoError,
		},
		{
			name: "get value from secret",
			sourceRef: &common.SourceRef{
				SecretKeyRef: &common.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "test-key",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"test-key": []byte("test-value"),
						},
					},
				).Build()
			},
			want:    "test-value",
			wantErr: require.NoError,
		},
		{
			name: "failed to get config map",
			sourceRef: &common.SourceRef{
				ConfigMapKeyRef: &common.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-configmap",
					},
					Key: "test-key",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			want:    "",
			wantErr: require.Error,
		},
		{
			name: "failed to get secret",
			sourceRef: &common.SourceRef{
				SecretKeyRef: &common.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "test-key",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			want:    "",
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetValueFromSourceRef(
				context.Background(),
				tt.sourceRef,
				"default",
				tt.k8sClient(t),
			)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
