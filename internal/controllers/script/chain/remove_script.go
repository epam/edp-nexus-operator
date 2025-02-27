package chain

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type RemoveScript struct {
	nexusScriptApiClient nexus.Script
}

func NewRemoveScript(nexusScriptApiClient nexus.Script) *RemoveScript {
	return &RemoveScript{nexusScriptApiClient: nexusScriptApiClient}
}

func (c *RemoveScript) ServeRequest(ctx context.Context, script *nexusApi.NexusScript) error {
	log := ctrl.LoggerFrom(ctx).WithValues("script_name", script.Spec.Name)
	log.Info("Start removing script")

	if err := c.nexusScriptApiClient.Delete(script.Spec.Name); err != nil {
		log.Info("Script doesn't exist, skipping removal")
	}

	log.Info("Script has been removed")

	return nil
}
