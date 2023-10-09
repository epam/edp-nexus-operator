package chain

import (
	"github.com/datadrivers/go-nexus-client/nexus3"
)

func MakeChain(nexusApiClient *nexus3.NexusClient) NexusRoleHandler {
	ch := &chain{}

	ch.Use(NewCreateRole(nexusApiClient.Security.Role))

	return ch
}
