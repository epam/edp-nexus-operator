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

func TestExecuteScript_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		script               *nexusApi.NexusScript
		nexusScriptApiClient func(t *testing.T) nexus.Script
		wantErr              require.ErrorAssertionFunc
	}{
		{
			name: "script executed successfully",
			script: &nexusApi.NexusScript{
				Spec: nexusApi.NexusScriptSpec{
					Name:    "test-script",
					Content: "println('test')",
					Payload: "test-payload",
					Execute: true,
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.On("RunWithPayload", "test-script", "test-payload").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to execute script",
			script: &nexusApi.NexusScript{
				Spec: nexusApi.NexusScriptSpec{
					Name:    "test-script",
					Content: "println('test')",
					Payload: "test-payload",
					Execute: true,
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.On("RunWithPayload", "test-script", "test-payload").
					Return(errors.New("failed to execute script"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to execute script")
			},
		},
		{
			name: "script execution is disabled",
			script: &nexusApi.NexusScript{
				Spec: nexusApi.NexusScriptSpec{
					Name:    "test-script",
					Content: "println('test')",
					Payload: "test-payload",
					Execute: false,
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.AssertNotCalled(t, "RunWithPayload", mock.Anything, mock.Anything)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "script has been executed already",
			script: &nexusApi.NexusScript{
				Spec: nexusApi.NexusScriptSpec{
					Name:    "test-script",
					Content: "println('test')",
					Payload: "test-payload",
					Execute: true,
				},
				Status: nexusApi.NexusScriptStatus{
					Executed: true,
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.AssertNotCalled(t, "RunWithPayload", mock.Anything, mock.Anything)

				return m
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewExecuteScript(tt.nexusScriptApiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.script)

			tt.wantErr(t, err)
		})
	}
}
