package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-nexus-operator/api/v1alpha1"
)

func TestGetHTTPAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data interface{}
		want bool
	}{
		{
			name: "proxy with authentication",
			data: &v1alpha1.GoProxyRepository{ProxySpec: v1alpha1.ProxySpec{
				HTTPClient: v1alpha1.HTTPClient{Authentication: &v1alpha1.HTTPClientAuthentication{}},
			}},
			want: true,
		},
		{
			name: "proxy without authentication",
			data: &v1alpha1.GoProxyRepository{},
		},
		{
			name: "maven proxy with authentication",
			data: &v1alpha1.MavenProxyRepository{HTTPClient: v1alpha1.HTTPClientWithPreemptiveAuth{
				Authentication: &v1alpha1.HTTPClientAuthenticationWithPreemptive{},
			}},
			want: true,
		},
		{
			name: "maven proxy without authentication",
			data: &v1alpha1.MavenProxyRepository{},
		},
		{
			name: "group repository",
			data: &v1alpha1.GoGroupRepository{},
		},
		{
			name: "hosted repository",
			data: &v1alpha1.MavenHostedRepository{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, GetHTTPAuth(tt.data) != nil)
		})
	}
}
