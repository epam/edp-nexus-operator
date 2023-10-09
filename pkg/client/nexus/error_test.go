package nexus

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsErrNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "err is nil",
			err:  nil,
			want: false,
		},
		{
			name: "err is not found",
			err:  errors.New("not found error"),
			want: true,
		},
		{
			name: "err is not found",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsErrNotFound(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
