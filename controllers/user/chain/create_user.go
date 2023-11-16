package chain

import (
	"context"
	"fmt"
	"strings"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

// CreateUser is a handler for creating user.
type CreateUser struct {
	nexusUserApiClient nexus.User
	client             client.Client
}

// NewCreateUser creates an instance of CreateUser handler.
func NewCreateUser(nexusUserApiClient nexus.User, k8sClient client.Client) *CreateUser {
	return &CreateUser{nexusUserApiClient: nexusUserApiClient, client: k8sClient}
}

// ServeRequest implements the logic of creating user.
func (c *CreateUser) ServeRequest(ctx context.Context, user *nexusApi.NexusUser) error {
	log := ctrl.LoggerFrom(ctx).WithValues("id", user.Spec.ID)
	log.Info("Start creating user")

	nexusUser, err := c.nexusUserApiClient.Get(user.Spec.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if nexusUser == nil {
		log.Info("User doesn't exist, creating new one")

		var pass string

		if pass, err = c.getSecretFromRef(ctx, user.Spec.Secret, user.Namespace); err != nil {
			return fmt.Errorf("failed to get password from secret: %w", err)
		}

		if err = c.nexusUserApiClient.Create(specToUser(&user.Spec, pass)); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		log.Info("User has been created")

		return nil
	}

	if userChanged(&user.Spec, nexusUser) {
		log.Info("Updating user")

		updateUserFields(&user.Spec, nexusUser)

		if err = c.nexusUserApiClient.Update(user.Spec.ID, *nexusUser); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		log.Info("User has been updated")

		return nil
	}

	log.Info("User unchanged, skip updating")

	return nil
}

func userChanged(spec *nexusApi.NexusUserSpec, nexusUser *security.User) bool {
	if spec.FirstName != nexusUser.FirstName ||
		spec.LastName != nexusUser.LastName ||
		spec.Email != nexusUser.EmailAddress ||
		spec.Status != nexusUser.Status ||
		!slices.Equal(spec.Roles, nexusUser.Roles) {
		return true
	}

	return false
}

func updateUserFields(spec *nexusApi.NexusUserSpec, user *security.User) {
	user.FirstName = spec.FirstName
	user.LastName = spec.LastName
	user.EmailAddress = spec.Email
	user.Status = spec.Status
	user.Roles = slices.Clone(spec.Roles)
}

func specToUser(spec *nexusApi.NexusUserSpec, password string) security.User {
	return security.User{
		UserID:       spec.ID,
		FirstName:    spec.FirstName,
		LastName:     spec.LastName,
		EmailAddress: spec.Email,
		Status:       spec.Status,
		Roles:        slices.Clone(spec.Roles),
		Password:     password,
	}
}

func (c *CreateUser) getSecretFromRef(ctx context.Context, refVal, secretNamespace string) (string, error) {
	if !hasSecretRef(refVal) {
		return "", fmt.Errorf("invalid config secret reference %s is not in format '$secretName:secretKey'", refVal)
	}

	ref := strings.Split(refVal[1:], ":")
	if len(ref) != 2 {
		return "", fmt.Errorf("invalid config secret  reference %s is not in format '$secretName:secretKey'", refVal)
	}

	secret := &corev1.Secret{}
	if err := c.client.Get(ctx, client.ObjectKey{
		Namespace: secretNamespace,
		Name:      ref[0],
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", ref[0], err)
	}

	secretVal, ok := secret.Data[ref[1]]
	if !ok {
		return "", fmt.Errorf("secret %s does not contain key %s", ref[0], ref[1])
	}

	return string(secretVal), nil
}

func hasSecretRef(val string) bool {
	return strings.HasPrefix(val, "$")
}
