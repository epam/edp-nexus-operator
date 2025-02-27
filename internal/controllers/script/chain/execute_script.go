package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type ExecuteScript struct {
	nexusScriptApiClient nexus.Script
}

func NewExecuteScript(nexusScriptApiClient nexus.Script) *ExecuteScript {
	return &ExecuteScript{nexusScriptApiClient: nexusScriptApiClient}
}

func (c *ExecuteScript) ServeRequest(ctx context.Context, script *nexusApi.NexusScript) error {
	log := ctrl.LoggerFrom(ctx).WithValues("script_name", script.Spec.Name)

	if !script.Spec.Execute {
		log.Info("Script execution is disabled")

		return nil
	}

	if script.Status.Executed {
		log.Info("Script has been already executed")

		return nil
	}

	if err := c.nexusScriptApiClient.RunWithPayload(script.Spec.Name, script.Spec.Payload); err != nil {
		return fmt.Errorf("failed to executed script: %w", err)
	}

	script.Status.Executed = true

	log.Info("Script has been executed")

	return nil
}
