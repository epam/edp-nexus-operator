package nexus

import (
	"context"

	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
)

type Child interface {
	OwnerName() string
	GetNamespace() string
}

func (n ServiceImpl) ClientForNexusChild(ctx context.Context, child Child) (*nexus.Client, error) {
	var nx v1alpha1.Nexus
	if err := n.client.Get(ctx, types.NamespacedName{
		Namespace: child.GetNamespace(),
		Name:      child.OwnerName(),
	}, &nx); err != nil {
		return nil, errors.Wrap(err, "unable to get nexus owner")
	}

	pwd, err := n.getNexusAdminPassword(nx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get nexus admin password")
	}

	u, err := n.getNexusRestApiUrl(nx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Nexus REST API URL")
	}

	return nexus.Init(u, nexusDefaultSpec.NexusDefaultAdminUser, pwd), nil
}
