package repository

import (
	"context"
	"fmt"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/controllers/repository/chain"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

const (
	nexusOperatorFinalizer = "edp.epam.com/finalizer"
	errorRequeueTime       = time.Second * 30
)

type apiClientProvider interface {
	GetNexusRepositoryClientFromNexusRef(ctx context.Context, namespace string, ref common.HasNexusRef) (*nexus.RepoClient, error)
}

// NexusRepositoryReconciler reconciles a NexusRepository object.
type NexusRepositoryReconciler struct {
	client            client.Client
	scheme            *runtime.Scheme
	apiClientProvider apiClientProvider
}

func NewNexusRepositoryReconciler(k8sClient client.Client, scheme *runtime.Scheme, apiClientProvider apiClientProvider) *NexusRepositoryReconciler {
	return &NexusRepositoryReconciler{client: k8sClient, scheme: scheme, apiClientProvider: apiClientProvider}
}

//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusrepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusrepositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusrepositories/finalizers,verbs=update
//+kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NexusRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling NexusRepository")

	repository := &nexusApi.NexusRepository{}
	if err := r.client.Get(ctx, req.NamespacedName, repository); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get NexusRepository: %w", err)
	}

	nexusApiClient, err := r.apiClientProvider.GetNexusRepositoryClientFromNexusRef(ctx, repository.Namespace, repository)
	if err != nil {
		log.Error(err, "An error has occurred while getting nexus api client")

		return ctrl.Result{
			RequeueAfter: errorRequeueTime,
		}, nil
	}

	if repository.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(repository, nexusOperatorFinalizer) {
			log.Info("Deleting NexusRepository")

			if err = chain.NewRemoveRepository(nexusApiClient).ServeRequest(ctx, repository); err != nil {
				log.Error(err, "An error has occurred while deleting NexusRepository")

				return ctrl.Result{
					RequeueAfter: errorRequeueTime,
				}, nil
			}

			controllerutil.RemoveFinalizer(repository, nexusOperatorFinalizer)

			if err = r.client.Update(ctx, repository); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update NexusRepository: %w", err)
			}
		}

		log.Info("NexusRepository has been deleted")

		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(repository, nexusOperatorFinalizer) {
		err = r.client.Update(ctx, repository)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update NexusRepository: %w", err)
		}
	}

	oldStatus := repository.Status

	if err = chain.NewCreateRepository(nexusApiClient).ServeRequest(ctx, repository); err != nil {
		log.Error(err, "An error has occurred while handling NexusRepository")

		repository.Status.Value = common.StatusError
		repository.Status.Error = err.Error()

		if err = r.updateNexusRepositoryStatus(ctx, repository, oldStatus); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: errorRequeueTime,
		}, nil
	}

	repository.Status.Value = common.StatusCreated
	repository.Status.Error = ""

	if err = r.updateNexusRepositoryStatus(ctx, repository, oldStatus); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciling NexusRepository has been finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NexusRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.NexusRepository{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to setup NexusRepository reconciler: %w", err)
	}

	return nil
}

func (r *NexusRepositoryReconciler) updateNexusRepositoryStatus(
	ctx context.Context,
	repository *nexusApi.NexusRepository,
	oldStatus nexusApi.NexusRepositoryStatus,
) error {
	if repository.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, repository); err != nil {
		return fmt.Errorf("failed to update NexusRepository status: %w", err)
	}

	return nil
}
