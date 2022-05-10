package nexus

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
)

type Child interface {
	OwnerName() string
	GetNamespace() string
}

func (s ServiceImpl) ClientForNexusChild(ctx context.Context, child Child) (*nexus.Client, error) {
	var nx v1.Nexus
	if err := s.client.Get(ctx, types.NamespacedName{
		Namespace: child.GetNamespace(),
		Name:      child.OwnerName(),
	}, &nx); err != nil {
		return nil, errors.Wrap(err, "unable to get nexus owner")
	}

	pwd, err := s.getNexusAdminPassword(nx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get nexus admin password")
	}

	u, err := s.getNexusRestApiUrl(nx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Nexus REST API URL")
	}

	return nexus.Init(u, nexusDefaultSpec.NexusDefaultAdminUser, pwd), nil
}
