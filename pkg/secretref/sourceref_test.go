package secretref

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-nexus-operator/api/common"
)

func TestGetValueFromSecretKeySelector(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "cred", Namespace: "ns"},
		Data:       map[string][]byte{"password": []byte("pw-1"), "empty": {}},
	}

	tests := []struct {
		name     string
		selector *common.SecretKeySelector
		want     string
		wantErr  string
	}{
		{
			name:     "nil selector",
			selector: nil,
			want:     "",
		},
		{
			name:     "value",
			selector: selectorOf("cred", "password"),
			want:     "pw-1",
		},
		{
			name:     "key present with empty value",
			selector: selectorOf("cred", "empty"),
			want:     "",
		},
		{
			name:     "missing key",
			selector: selectorOf("cred", "absent"),
			wantErr:  "secret cred does not contain key absent",
		},
		{
			name:     "missing secret",
			selector: selectorOf("absent", "password"),
			wantErr:  "unable to get secret absent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret.DeepCopy()).Build()

			got, err := GetValueFromSecretKeySelector(context.Background(), tt.selector, "ns", k8sClient)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Empty(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func selectorOf(name, key string) *common.SecretKeySelector {
	s := &common.SecretKeySelector{Key: key}
	s.Name = name

	return s
}
