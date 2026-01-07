package webhook

import (
	"context"
	"fmt"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-nexus-operator/pkg/helper"
)

// RegisterValidationWebHook registers a new webhook for validating CRD.
func RegisterValidationWebHook(ctx context.Context, mgr ctrl.Manager, namespace string) error {
	// for OLM installation we need to skip creating self-signed certificates. Certificates are managed by OLM.
	if os.Getenv("SETUP_SELF_SIGNED_CERTIFICATES") != "false" {
		if namespace == "" {
			return fmt.Errorf(
				"self-signed certificates can't be created in AllNamespaces mode, please specify %s",
				helper.WatchNamespaceEnvVar,
			)
		}

		// mgr.GetAPIReader() is used to read objects before cache is started.
		certService := NewCertService(mgr.GetAPIReader(), mgr.GetClient())
		if err := certService.PopulateCertificates(ctx, namespace); err != nil {
			return fmt.Errorf("failed to populate certificates: %w", err)
		}
	}

	nexusRepositoryWebHook := NewNexusRepositoryValidationWebhook()
	if err := nexusRepositoryWebHook.SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil
}
