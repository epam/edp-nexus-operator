package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestRemoveRole_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		role           *nexusApi.NexusRole
		nexusApiClient func(t *testing.T) nexus.Role
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name: "removing role successfully",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID: "role-id",
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Delete", "role-id").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "role doesn't exist, skipping removal",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID: "role-id",
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Delete", "role-id").
					Return(errors.New("not found"))

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to remove role",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID: "role-id",
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Delete", "role-id").
					Return(errors.New("failed to remove role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to remove role")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewRemoveRole(tt.nexusApiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.role)

			tt.wantErr(t, err)
		})
	}
}
