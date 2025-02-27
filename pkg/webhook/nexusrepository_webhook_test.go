package webhook

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

func TestNexusRepositoryValidationWebhook_ValidateCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		obj     runtime.Object
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "NexusRepository CR is valid",
			obj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "NexusRepository CR is invalid - multiple types",
			obj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
						Group: &nexusApi.GoGroupRepository{
							GroupSpec: nexusApi.GroupSpec{
								Name: "go-group",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "repository must have only one type")
			},
		},
		{
			name: "NexusRepository CR is invalid - multiple formats",
			obj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
					Maven: &nexusApi.MavenSpec{
						Proxy: &nexusApi.MavenProxyRepository{
							Name: "ma-proxy",
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "repository must have only one format")
			},
		},
		{
			name: "NexusRepository CR is invalid - no type",
			obj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "repository type is not specified")
			},
		},
		{
			name: "NexusRepository CR is invalid - no format",
			obj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "repository format is not specified")
			},
		},
		{
			name:    "wrong object given - skipping validation",
			obj:     &nexusApi.NexusUser{},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := admission.NewContextWithRequest(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				admission.Request{
					AdmissionRequest: v1.AdmissionRequest{
						Name:      "nexus-repository",
						Namespace: "default",
					},
				},
			)
			r := NewNexusRepositoryValidationWebhook()
			_, err := r.ValidateCreate(req, tt.obj)

			tt.wantErr(t, err)
		})
	}
}

func TestNexusRepositoryValidationWebhook_ValidateUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		oldObj  runtime.Object
		newObj  runtime.Object
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "NexusRepository CR is valid - no changes",
			oldObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			newObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "NexusRepository CR is valid - with changes",
			oldObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			newObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name:   "go-proxy",
								Online: true,
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "NexusRepository CR is invalid - no format",
			oldObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			newObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "repository format is not specified")
			},
		},
		{
			name: "NexusRepository CR is valid - type changed",
			oldObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			newObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Group: &nexusApi.GoGroupRepository{
							GroupSpec: nexusApi.GroupSpec{
								Name: "go-group",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "repository type proxy cannot be changed to another")
			},
		},
		{
			name: "NexusRepository CR is valid - format changed",
			oldObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Go: &nexusApi.GoSpec{
						Proxy: &nexusApi.GoProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "go-proxy",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			newObj: &nexusApi.NexusRepository{
				Spec: nexusApi.NexusRepositorySpec{
					Npm: &nexusApi.NpmSpec{
						Proxy: &nexusApi.NpmProxyRepository{
							ProxySpec: nexusApi.ProxySpec{
								Name: "npm-proxy",
							},
						},
					},
					NexusRef: common.NexusRef{
						Name: "nexus",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "repository format go cannot be changed to another")
			},
		},
		{
			name:    "wrong old object given - skipping validation",
			oldObj:  &nexusApi.NexusUser{},
			newObj:  &nexusApi.NexusRepository{},
			wantErr: require.NoError,
		},
		{
			name:    "wrong new object given - skipping validation",
			oldObj:  &nexusApi.NexusRepository{},
			newObj:  &nexusApi.NexusUser{},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := admission.NewContextWithRequest(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				admission.Request{
					AdmissionRequest: v1.AdmissionRequest{
						Name:      "nexus-repository",
						Namespace: "default",
					},
				},
			)
			r := &NexusRepositoryValidationWebhook{}
			_, err := r.ValidateUpdate(req, tt.oldObj, tt.newObj)

			tt.wantErr(t, err)
		})
	}
}
