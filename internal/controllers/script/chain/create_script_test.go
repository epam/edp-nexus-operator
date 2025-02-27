package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestCreateScript_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		script               *nexusApi.NexusScript
		nexusScriptApiClient func(t *testing.T) nexus.Script
		wantErr              require.ErrorAssertionFunc
	}{
		{
			name: "script doesn't exist, creating new one",
			script: &nexusApi.NexusScript{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-script",
					Namespace: "default",
				},
				Spec: nexusApi.NexusScriptSpec{
					Name:    "test-script",
					Content: "println('test')",
					Payload: "test-payload",
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.On("Get", "test-script").
					Return(nil, nexus.ErrNotFound)
				m.On("Create", &schema.Script{
					Name:    "test-script",
					Content: "println('test')",
					Type:    scriptTypeGroovy,
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "script exists, updating",
			script: &nexusApi.NexusScript{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-script",
					Namespace: "default",
				},
				Spec: nexusApi.NexusScriptSpec{
					Name:    "test-script",
					Content: "println('test')",
					Payload: "test-payload",
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.On("Get", "test-script").
					Return(&schema.Script{}, nil)
				m.On("Update", &schema.Script{
					Name:    "test-script",
					Content: "println('test')",
					Type:    scriptTypeGroovy,
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to update script",
			script: &nexusApi.NexusScript{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-script",
					Namespace: "default",
				},
				Spec: nexusApi.NexusScriptSpec{
					Name:    "test-script",
					Content: "println('test')",
					Payload: "test-payload",
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.On("Get", "test-script").
					Return(&schema.Script{}, nil)
				m.On("Update", &schema.Script{
					Name:    "test-script",
					Content: "println('test')",
					Type:    scriptTypeGroovy,
				}).Return(errors.New("failed to update script"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update script")
			},
		},
		{
			name: "failed to create script",
			script: &nexusApi.NexusScript{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-script",
					Namespace: "default",
				},
				Spec: nexusApi.NexusScriptSpec{
					Name:    "test-script",
					Content: "println('test')",
					Payload: "test-payload",
				},
			},
			nexusScriptApiClient: func(t *testing.T) nexus.Script {
				m := mocks.NewMockScript(t)

				m.On("Get", "test-script").
					Return(nil, nexus.ErrNotFound)
				m.On("Create", &schema.Script{
					Name:    "test-script",
					Content: "println('test')",
					Type:    scriptTypeGroovy,
				}).Return(errors.New("failed to create script"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create script")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewCreateScript(tt.nexusScriptApiClient(t))
			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.script)
			tt.wantErr(t, err)
		})
	}
}
