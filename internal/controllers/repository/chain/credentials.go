package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/helper"
)

// getRepoDataWithCredentials returns repository data where the HTTP client authentication
// Secret references are replaced by the values read from the referenced Secrets.
// The passed spec is never modified.
func getRepoDataWithCredentials(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	spec *nexusApi.NexusRepositorySpec,
) (*nexus.RepoData, error) {
	// Resolve on a copy so the credentials never reach the CR that the reconciler writes back.
	repoData, err := nexus.GetRepoData(spec.DeepCopy())
	if err != nil {
		return nil, fmt.Errorf("failed to get repository data: %w", err)
	}

	auth := nexus.GetHTTPAuth(repoData.Data)
	if auth == nil {
		return repoData, nil
	}

	usernameRef, passwordRef := auth.SecretKeyRefs()

	username, err := getSecretValue(ctx, k8sClient, namespace, usernameRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve httpClient username: %w", err)
	}

	password, err := getSecretValue(ctx, k8sClient, namespace, passwordRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve httpClient password: %w", err)
	}

	auth.SetResolvedCredentials(username, password)

	return repoData, nil
}

// getSecretValue returns the value selected by ref, or nil when ref is not set.
// A missing or empty value is an error, because Nexus accepts blank credentials and would replace working ones.
func getSecretValue(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	ref *common.SecretKeySelector,
) (*string, error) {
	if ref == nil {
		return nil, nil
	}

	value, err := helper.GetValueFromSourceRef(ctx, &common.SourceRef{SecretKeyRef: ref}, namespace, k8sClient)
	if err != nil {
		return nil, err
	}

	if value == "" {
		return nil, fmt.Errorf("key %q in secret %q is missing or empty", ref.Key, ref.Name)
	}

	return &value, nil
}
