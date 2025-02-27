package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	nexusclinet "github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type CheckConnection struct {
	nexusApiClient nexusclinet.User
}

func NewCheckConnection(nexusApiClient nexusclinet.User) *CheckConnection {
	return &CheckConnection{nexusApiClient: nexusApiClient}
}

func (h *CheckConnection) ServeRequest(ctx context.Context, nexus *nexusApi.Nexus) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start checking connection to nexus")

	// we can search for non-existent users to check the connection
	// if the user is not found, we will not get an error
	_, err := h.nexusApiClient.Get("user")
	if err != nil {
		return fmt.Errorf("failed to connect to nexus api: %w", err)
	}

	nexus.Status.Connected = true

	log.Info("Connection to nexus is established")

	return nil
}
