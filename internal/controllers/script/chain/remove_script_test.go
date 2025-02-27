package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestRemoveScript_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		script               *nexusApi.NexusScript
		nexusScriptApiClient func(t *testing.T) nexus.Script
		wantErr              require.ErrorAssertionFunc
	}{
		{
			name: "script doesn't exist, skipping removal",
			script: &nexusApi.NexusScript{
				Spec: nexusApi.NexusScriptSpec{
					Name: "test-script",
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.On("Delete", "test-script").
					Return(nexus.ErrNotFound)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "script exists, removing",
			script: &nexusApi.NexusScript{
				Spec: nexusApi.NexusScriptSpec{
					Name: "test-script",
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.On("Delete", "test-script").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewRemoveScript(tt.nexusScriptApiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.script)
			tt.wantErr(t, err)
		})
	}
}
