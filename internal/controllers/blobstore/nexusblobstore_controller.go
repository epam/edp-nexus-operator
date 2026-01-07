package blobstore

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
	"github.com/epam/edp-nexus-operator/internal/controllers/blobstore/chain"
)

// NexusBlobStoreReconciler reconciles a NexusBlobStore object.
type NexusBlobStoreReconciler struct {
	client            client.Client
	apiClientProvider controllers.ApiClientProvider
}

func NewNexusBlobStoreReconciler(k8sClient client.Client, apiClientProvider controllers.ApiClientProvider) *NexusBlobStoreReconciler {
	return &NexusBlobStoreReconciler{client: k8sClient, apiClientProvider: apiClientProvider}
}

// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusblobstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusblobstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusblobstores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NexusBlobStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling NexusBlobStore")

	store := &nexusApi.NexusBlobStore{}
	if err := r.client.Get(ctx, req.NamespacedName, store); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get NexusBlobStore: %w", err)
	}

	nexusApiClient, err := r.apiClientProvider.GetNexusApiClientFromNexusRef(ctx, store.Namespace, store)
	if err != nil {
		log.Error(err, "An error has occurred while getting nexus api client")

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	if store.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(store, controllers.NexusOperatorFinalizer) {
			if err = chain.NewRemoveBlobstore(nexusApiClient.BlobStore.File).ServeRequest(ctx, store); err != nil {
				log.Error(err, "An error has occurred while deleting NexusBlobStore")

				return ctrl.Result{
					RequeueAfter: controllers.ErrorRequeueTime,
				}, nil
			}

			controllerutil.RemoveFinalizer(store, controllers.NexusOperatorFinalizer)

			if err = r.client.Update(ctx, store); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update NexusBlobStore: %w", err)
			}
		}

		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(store, controllers.NexusOperatorFinalizer) {
		err = r.client.Update(ctx, store)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update NexusBlobStore: %w", err)
		}
	}

	oldStatus := store.Status

	if err = chain.NewCreateBlobStore(
		nexusApiClient.BlobStore.S3,
		nexusApiClient.BlobStore.File,
		r.client,
	).ServeRequest(ctx, store); err != nil {
		log.Error(err, "An error has occurred while handling NexusBlobStore")

		store.Status.Value = common.StatusError
		store.Status.Error = err.Error()

		if err = r.updateNexusBlobStoreStatus(ctx, store, oldStatus); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	store.Status.Value = common.StatusCreated
	store.Status.Error = ""

	if err = r.updateNexusBlobStoreStatus(ctx, store, oldStatus); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NexusBlobStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.NexusBlobStore{}).
		Complete(r)

	if err != nil {
		return fmt.Errorf("failed to setup NexusBlobStore controller: %w", err)
	}

	return nil
}

func (r *NexusBlobStoreReconciler) updateNexusBlobStoreStatus(
	ctx context.Context,
	store *nexusApi.NexusBlobStore,
	oldStatus nexusApi.NexusBlobStoreStatus,
) error {
	if store.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, store); err != nil {
		return fmt.Errorf("failed to update NexusBlobStore status: %w", err)
	}

	return nil
}
