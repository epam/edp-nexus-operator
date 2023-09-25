package chain

import (
	"github.com/datadrivers/go-nexus-client/nexus3"
)

func MakeChain(nexusApiClient *nexus3.NexusClient) NexusHandler {
	ch := &chain{}
	ch.Use(NewCheckConnection(nexusApiClient.Security.User))

	return ch
}
