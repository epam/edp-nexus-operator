package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus/mocks"
)

func TestCreateUser_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		user           *nexusApi.NexusUser
		k8sClient      func(t *testing.T) client.Client
		nexusApiClient func(t *testing.T) nexus.User
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name: "user doesn't exist, creating new one",
			user: &nexusApi.NexusUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user",
					Namespace: "default",
				},
				Spec: nexusApi.NexusUserSpec{
					ID:        "user-id",
					FirstName: "new-user-name",
					LastName:  "new-user-last-name",
					Email:     "new-user-email@gmail.com",
					Secret:    "$secret:secret-filed",
					Status:    nexusApi.UserStatusActive,
					Roles:     []string{"nx-admin"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()

				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: ctrl.ObjectMeta{
								Name:      "secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"secret-filed": []byte("user-password"),
							},
						},
					).
					Build()
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Get", "user-id").Return(nil, nil)

				m.On("Create", security.User{
					UserID:       "user-id",
					FirstName:    "new-user-name",
					LastName:     "new-user-last-name",
					EmailAddress: "new-user-email@gmail.com",
					Password:     "user-password",
					Status:       nexusApi.UserStatusActive,
					Roles:        []string{"nx-admin"},
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "user exist, updating user",
			user: &nexusApi.NexusUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user",
					Namespace: "default",
				},
				Spec: nexusApi.NexusUserSpec{
					ID:        "user-id",
					FirstName: "new-user-name",
					LastName:  "new-user-last-name",
					Email:     "new-user-email@gmail.com",
					Secret:    "$secret:secret-filed",
					Status:    nexusApi.UserStatusActive,
					Roles:     []string{"nx-admin"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Get", "user-id").Return(&security.User{
					UserID:       "user-id",
					FirstName:    "new-user-name1",
					LastName:     "new-user-last-name1",
					EmailAddress: "new-user-email@gmail1.com",
					Status:       nexusApi.UserStatusDisabled,
					Roles:        []string{},
					Password:     "user-password",
				}, nil)

				m.On("Update", "user-id", security.User{
					UserID:       "user-id",
					FirstName:    "new-user-name",
					LastName:     "new-user-last-name",
					EmailAddress: "new-user-email@gmail.com",
					Password:     "user-password",
					Status:       nexusApi.UserStatusActive,
					Roles:        []string{"nx-admin"},
				}).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "user secret not found",
			user: &nexusApi.NexusUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user",
					Namespace: "default",
				},
				Spec: nexusApi.NexusUserSpec{
					ID:        "user-id",
					FirstName: "new-user-name",
					LastName:  "new-user-last-name",
					Email:     "new-user-email@gmail.com",
					Secret:    "$secret:secret-filed",
					Status:    nexusApi.UserStatusActive,
					Roles:     []string{"nx-admin"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Get", "user-id").Return(nil, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get password from secret")
			},
		},
		{
			name: "failed to create user",
			user: &nexusApi.NexusUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user",
					Namespace: "default",
				},
				Spec: nexusApi.NexusUserSpec{
					ID:        "user-id",
					FirstName: "new-user-name",
					LastName:  "new-user-last-name",
					Email:     "new-user-email@gmail.com",
					Secret:    "$secret:secret-filed",
					Status:    nexusApi.UserStatusActive,
					Roles:     []string{"nx-admin"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()

				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: ctrl.ObjectMeta{
								Name:      "secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"secret-filed": []byte("user-password"),
							},
						},
					).
					Build()
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Get", "user-id").Return(nil, nil)

				m.On("Create", security.User{
					UserID:       "user-id",
					FirstName:    "new-user-name",
					LastName:     "new-user-last-name",
					EmailAddress: "new-user-email@gmail.com",
					Password:     "user-password",
					Status:       nexusApi.UserStatusActive,
					Roles:        []string{"nx-admin"},
				}).Return(errors.New("failed to create user"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create user")
			},
		},
		{
			name: "failed to update user",
			user: &nexusApi.NexusUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user",
					Namespace: "default",
				},
				Spec: nexusApi.NexusUserSpec{
					ID:        "user-id",
					FirstName: "new-user-name",
					LastName:  "new-user-last-name",
					Email:     "new-user-email@gmail.com",
					Secret:    "$secret:secret-filed",
					Status:    nexusApi.UserStatusActive,
					Roles:     []string{"nx-admin"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Get", "user-id").Return(&security.User{
					UserID:       "user-id",
					FirstName:    "new-user-name1",
					LastName:     "new-user-last-name1",
					EmailAddress: "new-user-email@gmail1.com",
					Status:       nexusApi.UserStatusDisabled,
					Roles:        []string{},
					Password:     "user-password",
				}, nil)

				m.On("Update", "user-id", security.User{
					UserID:       "user-id",
					FirstName:    "new-user-name",
					LastName:     "new-user-last-name",
					EmailAddress: "new-user-email@gmail.com",
					Password:     "user-password",
					Status:       nexusApi.UserStatusActive,
					Roles:        []string{"nx-admin"},
				}).Return(errors.New("failed to update user"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update user")
			},
		},
		{
			name: "failed get user",
			user: &nexusApi.NexusUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user",
					Namespace: "default",
				},
				Spec: nexusApi.NexusUserSpec{
					ID:        "user-id",
					FirstName: "new-user-name",
					LastName:  "new-user-last-name",
					Email:     "new-user-email@gmail.com",
					Secret:    "$secret:secret-filed",
					Status:    nexusApi.UserStatusActive,
					Roles:     []string{"nx-admin"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().Build()
			},
			nexusApiClient: func(t *testing.T) nexus.User {
				m := mocks.NewUser(t)

				m.On("Get", "user-id").
					Return(nil, errors.New("failed to get user"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get user")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewCreateUser(tt.nexusApiClient(t), tt.k8sClient(t))
			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.user)

			tt.wantErr(t, err)
		})
	}
}
