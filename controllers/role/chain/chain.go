package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

type NexusRoleHandler interface {
	ServeRequest(context.Context, *nexusApi.NexusRole) error
}

type chain struct {
	handlers []NexusRoleHandler
}

func (ch *chain) Use(handlers ...NexusRoleHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *chain) ServeRequest(ctx context.Context, s *nexusApi.NexusRole) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Starting NexusRole chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.ServeRequest(ctx, s)
		if err != nil {
			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of NexusRole has been finished")

	return nil
}
