package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

// nolint:lll // configuration
// +kubebuilder:webhook:path=/validate-edp-epam-com-v1alpha1-nexusrepository,mutating=false,failurePolicy=fail,sideEffects=None,groups=edp.epam.com,resources=nexusrepositories,verbs=create;update,versions=v1alpha1,name=vnexusrepository.kb.io,admissionReviewVersions=v1

// NexusRepositoryValidationWebhook is a webhook for validating NexusRepository CRD.
type NexusRepositoryValidationWebhook struct {
}

// NewNexusRepositoryValidationWebhook creates a new webhook for validating NexusRepository CR.
func NewNexusRepositoryValidationWebhook() *NexusRepositoryValidationWebhook {
	return &NexusRepositoryValidationWebhook{}
}

// SetupWebhookWithManager sets up the webhook with the manager for NexusRepository CR.
func (r *NexusRepositoryValidationWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&nexusApi.NexusRepository{}).
		WithValidator(r).
		Complete()
	if err != nil {
		return fmt.Errorf("failed to build NexusRepository validation webhook: %w", err)
	}

	return nil
}

var _ webhook.CustomValidator = &NexusRepositoryValidationWebhook{}

// ValidateCreate is a webhook for validating the creation of the NexusRepository CR.
func (*NexusRepositoryValidationWebhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (warnings admission.Warnings, err error) {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}

	log := ctrl.LoggerFrom(ctx).WithName("nexus_repository_validation_webhook").
		WithValues("name", req.Name, "namespace", req.Namespace)

	log.Info("Validate create")

	createdNexusRepository, ok := obj.(*nexusApi.NexusRepository)
	if !ok {
		log.Info("The wrong object given, skipping validation")

		return nil, nil
	}

	if err = validateCreate(&createdNexusRepository.Spec); err != nil {
		return nil, fmt.Errorf("object NexusRepository %s is invalid: %w", createdNexusRepository.Name, err)
	}

	return nil, nil
}

// ValidateUpdate is a webhook for validating the updating of the NexusRepository CR.
func (*NexusRepositoryValidationWebhook) ValidateUpdate(
	ctx context.Context,
	oldObj, newObj runtime.Object,
) (warnings admission.Warnings, err error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Validate update")

	oldNexusRepository, ok := oldObj.(*nexusApi.NexusRepository)
	if !ok {
		log.Info("The wrong object given, skipping validation")

		return nil, nil
	}

	updatedNexusRepository, ok := newObj.(*nexusApi.NexusRepository)
	if !ok {
		log.Info("The wrong object given, skipping validation")

		return nil, nil
	}

	if err = validateUpdate(&oldNexusRepository.Spec, &updatedNexusRepository.Spec); err != nil {
		return nil, fmt.Errorf("object NexusRepository %s is invalid: %w", updatedNexusRepository.Name, err)
	}

	return nil, nil
}

// ValidateDelete is a webhook for validating the deleting of the NexusRepository CR.
// It is skipped for now. Add kubebuilder:webhook:verbs=delete to enable it.
func (*NexusRepositoryValidationWebhook) ValidateDelete(
	_ context.Context,
	_ runtime.Object,
) (warnings admission.Warnings, err error) {
	return nil, nil
}
