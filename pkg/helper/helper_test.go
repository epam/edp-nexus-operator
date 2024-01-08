package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWatchNamespace(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T)
		want    string
	}{
		{
			name: "namespace is set",
			prepare: func(t *testing.T) {
				t.Setenv(WatchNamespaceEnvVar, "test")
			},
			want: "test",
		},
		{
			name:    "namespace is  not set",
			prepare: func(t *testing.T) {},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			got := GetWatchNamespace()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetDebugMode(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T)
		want    bool
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "debug mode is set",
			prepare: func(t *testing.T) {
				t.Setenv(debugModeEnvVar, "true")
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name:    "debug mode is  not set",
			prepare: func(t *testing.T) {},
			want:    false,
			wantErr: require.NoError,
		},
		{
			name: "debug mode is not a bool",
			prepare: func(t *testing.T) {
				t.Setenv(debugModeEnvVar, "not-a-bool")
			},
			want: false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid syntax")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			got, err := GetDebugMode()

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
