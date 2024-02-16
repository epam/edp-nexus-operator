package nexus

import (
	"context"
	"fmt"

	"github.com/datadrivers/go-nexus-client/nexus3"
	nexus3client "github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-nexus-operator/api/common"
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
	secret, err := p.getNexusSecret(ctx, nexus)
	if err != nil {
		return nil, err
	}

	return nexus3.NewClient(nexus3client.Config{
		URL:      nexus.Spec.Url,
		Username: string(secret.Data["user"]),
		Password: string(secret.Data["password"]),
	}), nil
}

func (p *ApiClientProvider) GetNexusApiClientFromNexusRef(ctx context.Context, namespace string, ref common.HasNexusRef) (*nexus3.NexusClient, error) {
	nexus := &nexusApi.Nexus{}
	if err := p.k8sClient.Get(ctx, types.NamespacedName{
		Name:      ref.GetNexusRef().Name,
		Namespace: namespace,
	}, nexus); err != nil {
		return nil, fmt.Errorf("failed to get nexus instance: %w", err)
	}

	return p.GetNexusApiClientFromNexus(ctx, nexus)
}

func (p *ApiClientProvider) GetNexusRepositoryClientFromNexusRef(ctx context.Context, namespace string, ref common.HasNexusRef) (*RepoClient, error) {
	nexus := &nexusApi.Nexus{}
	if err := p.k8sClient.Get(ctx, types.NamespacedName{
		Name:      ref.GetNexusRef().Name,
		Namespace: namespace,
	}, nexus); err != nil {
		return nil, fmt.Errorf("failed to get nexus instance: %w", err)
	}

	secret, err := p.getNexusSecret(ctx, nexus)
	if err != nil {
		return nil, err
	}

	return NewRepoClient(ClientConfig{
		BaseURL:  nexus.Spec.Url,
		UserName: string(secret.Data["user"]),
		Password: string(secret.Data["password"]),
	}), nil
}

func (p *ApiClientProvider) GetNexusNexusCleanupPolicyClientFromNexusRef(
	ctx context.Context,
	namespace string,
	ref common.HasNexusRef,
) (*NexusCleanupPolicyClient, error) {
	nexus := &nexusApi.Nexus{}
	if err := p.k8sClient.Get(ctx, types.NamespacedName{
		Name:      ref.GetNexusRef().Name,
		Namespace: namespace,
	}, nexus); err != nil {
		return nil, fmt.Errorf("failed to get nexus instance: %w", err)
	}

	secret, err := p.getNexusSecret(ctx, nexus)
	if err != nil {
		return nil, err
	}

	return NewNexusCleanupPolicyClient(ClientConfig{
		BaseURL:  nexus.Spec.Url,
		UserName: string(secret.Data["user"]),
		Password: string(secret.Data["password"]),
	}), nil
}

func (p *ApiClientProvider) getNexusSecret(ctx context.Context, nexus *nexusApi.Nexus) (corev1.Secret, error) {
	secret := corev1.Secret{}
	if err := p.k8sClient.Get(ctx, types.NamespacedName{
		Name:      nexus.Spec.Secret,
		Namespace: nexus.Namespace,
	}, &secret); err != nil {
		return corev1.Secret{}, fmt.Errorf("failed to get nexus secret: %w", err)
	}

	if secret.Data["user"] == nil {
		return corev1.Secret{}, fmt.Errorf("nexus secret doesn't contain user")
	}

	if secret.Data["password"] == nil {
		return corev1.Secret{}, fmt.Errorf("nexus secret doesn't contain password")
	}

	return secret, nil
}
