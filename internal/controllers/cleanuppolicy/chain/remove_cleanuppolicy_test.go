package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestRemoveCleanupPolicy_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		policy    *nexusApi.NexusCleanupPolicy
		apiClient func(t *testing.T) nexus.NexusCleanupPolicyManager
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "cleanup policy successfully removed",
			policy: &nexusApi.NexusCleanupPolicy{
				Spec: nexusApi.NexusCleanupPolicySpec{
					Name: "test-policy",
				},
			},
			apiClient: func(t *testing.T) nexus.NexusCleanupPolicyManager {
				m := mocks.NewMockNexusCleanupPolicyManager(t)

				m.On("Delete", mock.Anything, "test-policy").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "cleanup policy doesn't exist, skipping removal",
			policy: &nexusApi.NexusCleanupPolicy{
				Spec: nexusApi.NexusCleanupPolicySpec{
					Name: "test-policy",
				},
			},
			apiClient: func(t *testing.T) nexus.NexusCleanupPolicyManager {
				m := mocks.NewMockNexusCleanupPolicyManager(t)

				m.On("Delete", mock.Anything, "test-policy").
					Return(nexus.ErrNotFound)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to remove cleanup policy",
			policy: &nexusApi.NexusCleanupPolicy{
				Spec: nexusApi.NexusCleanupPolicySpec{
					Name: "test-policy",
				},
			},
			apiClient: func(t *testing.T) nexus.NexusCleanupPolicyManager {
				m := mocks.NewMockNexusCleanupPolicyManager(t)

				m.On("Delete", mock.Anything, "test-policy").
					Return(errors.New("test error"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete cleanup policy")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewRemoveCleanupPolicy(tt.apiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.policy)

			tt.wantErr(t, err)
		})
	}
}
