package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestCreateNexusCleanupPolicy_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		policy    *nexusApi.NexusCleanupPolicy
		apiClient func(t *testing.T) nexus.NexusCleanupPolicyManager
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "cleanup policy doesn't exist, creating new one",
			policy: &nexusApi.NexusCleanupPolicy{
				Spec: nexusApi.NexusCleanupPolicySpec{
					Name:        "test-policy",
					Format:      "go",
					Description: "test policy description",
					Criteria: nexusApi.Criteria{
						LastBlobUpdated: 100,
					},
				},
			},
			apiClient: func(t *testing.T) nexus.NexusCleanupPolicyManager {
				m := mocks.NewMockNexusCleanupPolicyManager(t)

				m.On("Get", mock.Anything, "test-policy").
					Return(nil, nexus.ErrNotFound)
				m.On("Create", mock.Anything, &nexus.NexusCleanupPolicy{
					Name:                    "test-policy",
					Format:                  "go",
					Notes:                   "test policy description",
					CriteriaLastBlobUpdated: ptr.To(100),
				}).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "cleanup policy exists, updating",
			policy: &nexusApi.NexusCleanupPolicy{
				Spec: nexusApi.NexusCleanupPolicySpec{
					Name:        "test-policy",
					Format:      "go",
					Description: "test policy description",
				},
			},
			apiClient: func(t *testing.T) nexus.NexusCleanupPolicyManager {
				m := mocks.NewMockNexusCleanupPolicyManager(t)

				m.On("Get", mock.Anything, "test-policy").
					Return(&nexus.NexusCleanupPolicy{
						Name:   "test-policy",
						Format: "go",
						Notes:  "test",
					}, nil)
				m.On("Update", mock.Anything, "test-policy", &nexus.NexusCleanupPolicy{
					Name:   "test-policy",
					Format: "go",
					Notes:  "test policy description",
				}).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to update cleanup policy",
			policy: &nexusApi.NexusCleanupPolicy{
				Spec: nexusApi.NexusCleanupPolicySpec{
					Name:        "test-policy",
					Format:      "go",
					Description: "test policy description",
				},
			},
			apiClient: func(t *testing.T) nexus.NexusCleanupPolicyManager {
				m := mocks.NewMockNexusCleanupPolicyManager(t)

				m.On("Get", mock.Anything, "test-policy").
					Return(&nexus.NexusCleanupPolicy{
						Name:   "test-policy",
						Format: "go",
						Notes:  "test",
					}, nil)
				m.On("Update", mock.Anything, "test-policy", &nexus.NexusCleanupPolicy{
					Name:   "test-policy",
					Format: "go",
					Notes:  "test policy description",
				}).
					Return(errors.New("failed to update cleanup policy"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update cleanup policy")
			},
		},
		{
			name: "failed to create cleanup policy",
			policy: &nexusApi.NexusCleanupPolicy{
				Spec: nexusApi.NexusCleanupPolicySpec{
					Name:        "test-policy",
					Format:      "go",
					Description: "test policy description",
					Criteria: nexusApi.Criteria{
						LastBlobUpdated: 100,
					},
				},
			},
			apiClient: func(t *testing.T) nexus.NexusCleanupPolicyManager {
				m := mocks.NewMockNexusCleanupPolicyManager(t)

				m.On("Get", mock.Anything, "test-policy").
					Return(nil, nexus.ErrNotFound)
				m.On("Create", mock.Anything, &nexus.NexusCleanupPolicy{
					Name:                    "test-policy",
					Format:                  "go",
					Notes:                   "test policy description",
					CriteriaLastBlobUpdated: ptr.To(100),
				}).
					Return(errors.New("failed to create cleanup policy"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create cleanup policy")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewCreateNexusCleanupPolicy(tt.apiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.policy)

			tt.wantErr(t, err)
		})
	}
}
