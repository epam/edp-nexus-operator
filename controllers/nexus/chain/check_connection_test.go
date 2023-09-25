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

func TestCheckConnection_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		nexus          *nexusApi.Nexus
		nexusApiClient func(t *testing.T) nexus.User
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name:  "connection is established",
			nexus: &nexusApi.Nexus{},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Get", "user").Return(nil, nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name:  "failed to connect",
			nexus: &nexusApi.Nexus{},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Get", "user").Return(nil, errors.New("failed to connect"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to connect")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewCheckConnection(tt.nexusApiClient(t))
			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.nexus)

			tt.wantErr(t, err)
		})
	}
}
