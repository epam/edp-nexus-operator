package nexus

import (
	"context"
	"fmt"

	"github.com/datadrivers/go-nexus-client/nexus3"
	nexus3client "github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

// ApiClientProvider is a struct for providing nexus api client.
type ApiClientProvider struct {
	k8sClient client.Client
}

// NewApiClientProvider returns a new instance of ApiClientProvider.
func NewApiClientProvider(k8sClient client.Client) *ApiClientProvider {
	return &ApiClientProvider{k8sClient: k8sClient}
}

// GetNexusApiClientFromNexus returns nexus api client from Nexus CR.
func (p *ApiClientProvider) GetNexusApiClientFromNexus(ctx context.Context, nexus *nexusApi.Nexus) (*nexus3.NexusClient, error) {
	secret := corev1.Secret{}
	if err := p.k8sClient.Get(ctx, types.NamespacedName{
		Name:      nexus.Spec.Secret,
		Namespace: nexus.Namespace,
	}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get nexus secret: %w", err)
	}

	if secret.Data["user"] == nil {
		return nil, fmt.Errorf("nexus secret doesn't contain user")
	}

	if secret.Data["password"] == nil {
		return nil, fmt.Errorf("nexus secret doesn't contain password")
	}

	return nexus3.NewClient(nexus3client.Config{
		URL:      nexus.Spec.Url,
		Username: string(secret.Data["user"]),
		Password: string(secret.Data["password"]),
	}), nil
}
