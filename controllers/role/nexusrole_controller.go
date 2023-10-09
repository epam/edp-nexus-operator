package role

import (
	"context"
	"fmt"
	"time"

	"github.com/datadrivers/go-nexus-client/nexus3"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/controllers/role/chain"
)

const (
	nexusOperatorFinalizer = "edp.epam.com/finalizer"
	errorRequeueTime       = time.Second * 30
)

type apiClientProvider interface {
	GetNexusApiClientFromNexusRef(ctx context.Context, namespace string, ref common.HasNexusRef) (*nexus3.NexusClient, error)
}

// NexusRoleReconciler reconciles a NexusRole object.
type NexusRoleReconciler struct {
	client            client.Client
	scheme            *runtime.Scheme
	apiClientProvider apiClientProvider
}

func NewNexusRoleReconciler(k8sClient client.Client, scheme *runtime.Scheme, apiClientProvider apiClientProvider) *NexusRoleReconciler {
	return &NexusRoleReconciler{client: k8sClient, scheme: scheme, apiClientProvider: apiClientProvider}
}

//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusroles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusroles/finalizers,verbs=update
//+kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NexusRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling NexusRole")

	role := &nexusApi.NexusRole{}
	if err := r.client.Get(ctx, req.NamespacedName, role); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get NexusRole: %w", err)
	}

	nexusApiClient, err := r.apiClientProvider.GetNexusApiClientFromNexusRef(ctx, role.Namespace, role)
	if err != nil {
		log.Error(err, "An error has occurred while getting nexus api client")

		return ctrl.Result{
			RequeueAfter: errorRequeueTime,
		}, nil
	}

	if role.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(role, nexusOperatorFinalizer) {
			if err = chain.NewRemoveRole(nexusApiClient.Security.Role).ServeRequest(ctx, role); err != nil {
				log.Error(err, "An error has occurred while deleting NexusRole")

				return ctrl.Result{
					RequeueAfter: errorRequeueTime,
				}, nil
			}

			controllerutil.RemoveFinalizer(role, nexusOperatorFinalizer)

			if err = r.client.Update(ctx, role); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update NexusRole: %w", err)
			}
		}

		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(role, nexusOperatorFinalizer) {
		err = r.client.Update(ctx, role)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update NexusRole: %w", err)
		}
	}

	oldStatus := role.Status

	if err = chain.MakeChain(nexusApiClient).ServeRequest(ctx, role); err != nil {
		log.Error(err, "An error has occurred while handling NexusRole")

		role.Status.Value = "error"
		role.Status.Error = err.Error()

		if err = r.updateNexusRoleStatus(ctx, role, oldStatus); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: errorRequeueTime,
		}, nil
	}

	role.Status.Value = common.StatusCreated
	role.Status.Error = ""

	if err = r.updateNexusRoleStatus(ctx, role, oldStatus); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NexusRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.NexusRole{}).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	return nil
}

func (r *NexusRoleReconciler) updateNexusRoleStatus(
	ctx context.Context,
	role *nexusApi.NexusRole,
	oldStatus nexusApi.NexusRoleStatus,
) error {
	if role.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, role); err != nil {
		return fmt.Errorf("failed to update NexusRole status: %w", err)
	}

	return nil
}
