package nexus

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	nexusApi "github.com/epam/edp-nexus-operator/v2/api/v1"
	"github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
)

type Child interface {
	OwnerName() string
	GetNamespace() string
}

func (s ServiceImpl) ClientForNexusChild(ctx context.Context, child Child) (*nexus.Client, error) {
	nx := new(nexusApi.Nexus)
	if err := s.client.Get(ctx, types.NamespacedName{
		Namespace: child.GetNamespace(),
		Name:      child.OwnerName(),
	}, nx); err != nil {
		return nil, fmt.Errorf("failed to get nexus owner: %w", err)
	}

	pwd, err := s.getNexusAdminPassword(nx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nexus admin password: %w", err)
	}

	u, err := s.getNexusRestApiUrl(nx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Nexus REST API URL: %w", err)
	}

	return nexus.Init(u, nexusDefaultSpec.NexusDefaultAdminUser, pwd), nil
}
