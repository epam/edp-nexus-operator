package cleanuppolicy

import (
	"context"
	"fmt"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/internal/controllers"
	"github.com/epam/edp-nexus-operator/internal/controllers/cleanuppolicy/chain"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

type apiClientProvider interface {
	GetNexusNexusCleanupPolicyClientFromNexusRef(
		ctx context.Context,
		namespace string,
		ref common.HasNexusRef,
	) (*nexus.NexusCleanupPolicyClient, error)
}

// NexusCleanupPolicyReconciler reconciles a NexusCleanupPolicy object.
type NexusCleanupPolicyReconciler struct {
	client            client.Client
	apiClientProvider apiClientProvider
}

func NewNexusCleanupPolicyReconciler(k8sClient client.Client, apiClientProvider apiClientProvider) *NexusCleanupPolicyReconciler {
	return &NexusCleanupPolicyReconciler{client: k8sClient, apiClientProvider: apiClientProvider}
}

// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexuscleanuppolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexuscleanuppolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexuscleanuppolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NexusCleanupPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling NexusCleanupPolicy")

	policy := &nexusApi.NexusCleanupPolicy{}
	if err := r.client.Get(ctx, req.NamespacedName, policy); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get NexusCleanupPolicy: %w", err)
	}

	nexusApiClient, err := r.apiClientProvider.GetNexusNexusCleanupPolicyClientFromNexusRef(ctx, policy.Namespace, policy)
	if err != nil {
		log.Error(err, "An error has occurred while getting nexus api client")

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	if policy.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(policy, controllers.NexusOperatorFinalizer) {
			if err = chain.NewRemoveCleanupPolicy(nexusApiClient).ServeRequest(ctx, policy); err != nil {
				log.Error(err, "An error has occurred while deleting NexusCleanupPolicy")

				return ctrl.Result{
					RequeueAfter: controllers.ErrorRequeueTime,
				}, nil
			}

			controllerutil.RemoveFinalizer(policy, controllers.NexusOperatorFinalizer)

			if err = r.client.Update(ctx, policy); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update NexusCleanupPolicy: %w", err)
			}
		}

		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(policy, controllers.NexusOperatorFinalizer) {
		err = r.client.Update(ctx, policy)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update NexusCleanupPolicy: %w", err)
		}
	}

	oldStatus := policy.Status

	if err = chain.NewCreateNexusCleanupPolicy(nexusApiClient).ServeRequest(ctx, policy); err != nil {
		log.Error(err, "An error has occurred while handling NexusCleanupPolicy")

		policy.Status.Value = common.StatusError
		policy.Status.Error = err.Error()

		if err = r.updateNexusCleanupPolicyStatus(ctx, policy, oldStatus); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	policy.Status.Value = common.StatusCreated
	policy.Status.Error = ""

	if err = r.updateNexusCleanupPolicyStatus(ctx, policy, oldStatus); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NexusCleanupPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.NexusCleanupPolicy{}).
		Complete(r)

	if err != nil {
		return fmt.Errorf("failed to setup NexusCleanupPolicy controller: %w", err)
	}

	return nil
}

func (r *NexusCleanupPolicyReconciler) updateNexusCleanupPolicyStatus(
	ctx context.Context,
	policy *nexusApi.NexusCleanupPolicy,
	oldStatus nexusApi.NexusCleanupPolicyStatus,
) error {
	if policy.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, policy); err != nil {
		return fmt.Errorf("failed to update NexusCleanupPolicy status: %w", err)
	}

	return nil
}
