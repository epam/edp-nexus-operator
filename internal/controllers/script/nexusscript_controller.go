package script

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
	"github.com/epam/edp-nexus-operator/internal/controllers/script/chain"
)

// NexusScriptReconciler reconciles a NexusScript object.
type NexusScriptReconciler struct {
	k8sclient         client.Client
	apiClientProvider controllers.ApiClientProvider
}

func NewNexusScriptReconciler(k8sclient client.Client, apiClientProvider controllers.ApiClientProvider) *NexusScriptReconciler {
	return &NexusScriptReconciler{k8sclient: k8sclient, apiClientProvider: apiClientProvider}
}

//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusscripts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusscripts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusscripts/finalizers,verbs=update
//+kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NexusScriptReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling NexusScript")

	script := &nexusApi.NexusScript{}
	if err := r.k8sclient.Get(ctx, req.NamespacedName, script); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get NexusScript: %w", err)
	}

	nexusApiClient, err := r.apiClientProvider.GetNexusApiClientFromNexusRef(ctx, script.Namespace, script)
	if err != nil {
		log.Error(err, "An error has occurred while getting nexus api k8sclient")

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	if script.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(script, controllers.NexusOperatorFinalizer) {
			if err = chain.NewRemoveScript(nexusApiClient.Script).ServeRequest(ctx, script); err != nil {
				log.Error(err, "An error has occurred while deleting NexusScript")

				return ctrl.Result{
					RequeueAfter: controllers.ErrorRequeueTime,
				}, nil
			}

			controllerutil.RemoveFinalizer(script, controllers.NexusOperatorFinalizer)

			if err = r.k8sclient.Update(ctx, script); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update NexusScript: %w", err)
			}
		}

		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(script, controllers.NexusOperatorFinalizer) {
		err = r.k8sclient.Update(ctx, script)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update NexusScript: %w", err)
		}
	}

	oldStatus := script.Status

	if err = chain.CreateChain(nexusApiClient.Script).ServeRequest(ctx, script); err != nil {
		log.Error(err, "An error has occurred while handling NexusScript")

		script.Status.Value = common.StatusError
		script.Status.Error = err.Error()

		if err = r.updateNexusScriptStatus(ctx, script, oldStatus); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	script.Status.Value = common.StatusCreated
	script.Status.Error = ""

	if err = r.updateNexusScriptStatus(ctx, script, oldStatus); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NexusScriptReconciler) updateNexusScriptStatus(
	ctx context.Context,
	script *nexusApi.NexusScript,
	oldStatus nexusApi.NexusScriptStatus,
) error {
	if script.Status == oldStatus {
		return nil
	}

	if err := r.k8sclient.Status().Update(ctx, script); err != nil {
		return fmt.Errorf("failed to update NexusScript status: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NexusScriptReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.NexusScript{}).
		Complete(r)

	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	return nil
}
