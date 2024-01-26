package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

type scriptHandler interface {
	ServeRequest(ctx context.Context, script *nexusApi.NexusScript) error
}

type Chain struct {
	handlers []scriptHandler
}

func (ch *Chain) Use(handlers ...scriptHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) ServeRequest(ctx context.Context, script *nexusApi.NexusScript) error {
	log := ctrl.LoggerFrom(ctx).WithValues("script_name", script.Spec.Name)

	log.Info("Starting script Chain", "script_name", script.Spec.Name)

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.ServeRequest(ctx, script)
		if err != nil {
			log.Info("Script Chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of script has been finished")

	return nil
}
