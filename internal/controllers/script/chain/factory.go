package chain

import (
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

func CreateChain(nexusScriptApiClient nexus.Script) *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateScript(nexusScriptApiClient),
		NewExecuteScript(nexusScriptApiClient),
	)

	return ch
}
