package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestCreateRole_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		role           *nexusApi.NexusRole
		nexusApiClient func(t *testing.T) nexus.Role
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name: "role doesn't exist, creating new one",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID:          "role-id",
					Name:        "role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Get", "role-id").Return(nil, errors.New("not found"))

				m.On("Create", security.Role{
					ID:          "role-id",
					Name:        "role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "role exists, updating it",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID:          "role-id",
					Name:        "new-role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Get", "role-id").Return(&security.Role{
					ID:          "role-id",
					Name:        "role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				}, nil)

				m.On("Update", "role-id", security.Role{
					ID:          "role-id",
					Name:        "new-role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "role exists, no changes",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID:          "role-id",
					Name:        "role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Get", "role-id").Return(&security.Role{
					ID:          "role-id",
					Name:        "role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				}, nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to update role",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID:          "role-id",
					Name:        "new-role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Get", "role-id").Return(&security.Role{
					ID:          "role-id",
					Name:        "role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				}, nil)

				m.On("Update", "role-id", security.Role{
					ID:          "role-id",
					Name:        "new-role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				}).Return(errors.New("failed to update role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update role")
			},
		},
		{
			name: "failed to get role",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID:          "role-id",
					Name:        "new-role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Get", "role-id").Return(nil, errors.New("failed to get role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get role")
			},
		},
		{
			name: "failed to create role",
			role: &nexusApi.NexusRole{
				Spec: nexusApi.NexusRoleSpec{
					ID:          "role-id",
					Name:        "role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				},
			},
			nexusApiClient: func(t *testing.T) nexus.Role {
				m := mocks.NewMockRole(t)

				m.On("Get", "role-id").Return(nil, errors.New("not found"))

				m.On("Create", security.Role{
					ID:          "role-id",
					Name:        "role-name",
					Description: "role-description",
					Privileges:  []string{"privilege"},
				}).Return(errors.New("failed to create role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create role")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewCreateRole(tt.nexusApiClient(t))
			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.role)

			tt.wantErr(t, err)
		})
	}
}
