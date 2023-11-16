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

func TestRemoveUser_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		user           *nexusApi.NexusUser
		nexusApiClient func(t *testing.T) nexus.User
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name: "removing user successfully",
			user: &nexusApi.NexusUser{
				Spec: nexusApi.NexusUserSpec{
					ID: "user-id",
				},
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Delete", "user-id").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "user doesn't exist, skipping removal",
			user: &nexusApi.NexusUser{
				Spec: nexusApi.NexusUserSpec{
					ID: "user-id",
				},
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Delete", "user-id").
					Return(errors.New("not found"))

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to remove user",
			user: &nexusApi.NexusUser{
				Spec: nexusApi.NexusUserSpec{
					ID: "user-id",
				},
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Delete", "user-id").
					Return(errors.New("failed to remove user"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to remove user")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewRemoveUser(tt.nexusApiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.user)

			tt.wantErr(t, err)
		})
	}
}
