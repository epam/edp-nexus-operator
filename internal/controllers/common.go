package controllers

import (
	"context"
	"time"

	"github.com/datadrivers/go-nexus-client/nexus3"

	"github.com/epam/edp-nexus-operator/api/common"
)

const (
	NexusOperatorFinalizer = "edp.epam.com/finalizer"
	ErrorRequeueTime       = time.Second * 30
)

type ApiClientProvider interface {
	GetNexusApiClientFromNexusRef(ctx context.Context, namespace string, ref common.HasNexusRef) (*nexus3.NexusClient, error)
}
