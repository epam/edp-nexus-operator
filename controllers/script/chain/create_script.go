package chain

import (
	"context"
	"fmt"

	"github.com/datadrivers/go-nexus-client/nexus3/schema"
	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

const (
	scriptTypeGroovy = "groovy"
)

type CreateScript struct {
	nexusScriptApiClient nexus.Script
}

func NewCreateScript(nexusScriptApiClient nexus.Script) *CreateScript {
	return &CreateScript{nexusScriptApiClient: nexusScriptApiClient}
}

func (c *CreateScript) ServeRequest(ctx context.Context, script *nexusApi.NexusScript) error {
	log := ctrl.LoggerFrom(ctx).WithValues("script_name", script.Spec.Name)
	log.Info("Start creating script")

	_, getScriptErr := c.nexusScriptApiClient.Get(script.Spec.Name)
	if getScriptErr != nil {
		log.Info("Script doesn't exist, creating new one")

		if err := c.nexusScriptApiClient.Create(specToScript(&script.Spec)); err != nil {
			return fmt.Errorf("failed to create script: %w", err)
		}

		log.Info("Script has been created")
	}

	if getScriptErr == nil {
		log.Info("Updating script")

		if err := c.nexusScriptApiClient.Update(specToScript(&script.Spec)); err != nil {
			return fmt.Errorf("failed to update script: %w", err)
		}

		log.Info("Script has been updated")
	}

	return nil
}

func specToScript(spec *nexusApi.NexusScriptSpec) *schema.Script {
	return &schema.Script{
		Name:    spec.Name,
		Type:    scriptTypeGroovy,
		Content: spec.Content,
	}
}
