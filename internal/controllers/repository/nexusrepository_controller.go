package repository

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/internal/controllers"
	"github.com/epam/edp-nexus-operator/internal/controllers/repository/chain"
	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
)

// ctrlLog instance can be used when we can't get logger from context.
var ctrlLog = ctrl.Log.WithName("nexus_repository_ctrl")

type apiClientProvider interface {
	GetNexusRepositoryClientFromNexusRef(ctx context.Context, namespace string, ref common.HasNexusRef) (*nexus.RepoClient, error)
}

// NexusRepositoryReconciler reconciles a NexusRepository object.
type NexusRepositoryReconciler struct {
	client            client.Client
	apiClientProvider apiClientProvider
}

func NewNexusRepositoryReconciler(k8sClient client.Client, apiClientProvider apiClientProvider) *NexusRepositoryReconciler {
	return &NexusRepositoryReconciler{client: k8sClient, apiClientProvider: apiClientProvider}
}

// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusrepositories/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

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
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	if repository.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(repository, controllers.NexusOperatorFinalizer) {
			log.Info("Deleting NexusRepository")

			if err = chain.NewRemoveRepository(nexusApiClient).ServeRequest(ctx, repository); err != nil {
				log.Error(err, "An error has occurred while deleting NexusRepository")

				return ctrl.Result{
					RequeueAfter: controllers.ErrorRequeueTime,
				}, nil
			}

			controllerutil.RemoveFinalizer(repository, controllers.NexusOperatorFinalizer)

			if err = r.client.Update(ctx, repository); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update NexusRepository: %w", err)
			}
		}

		log.Info("NexusRepository has been deleted")

		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(repository, controllers.NexusOperatorFinalizer) {
		err = r.client.Update(ctx, repository)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update NexusRepository: %w", err)
		}
	}

	oldStatus := repository.Status

	if err = chain.NewCreateRepository(nexusApiClient, r.client).ServeRequest(ctx, repository); err != nil {
		log.Error(err, "An error has occurred while handling NexusRepository")

		repository.Status.Value = common.StatusError
		repository.Status.Error = err.Error()

		if err = r.updateNexusRepositoryStatus(ctx, repository, oldStatus); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
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
		Watches(
			// Watch Secrets to re-apply repository HTTP client credentials when they change. Deleted Secrets are ignored.
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToNexusRepository),
			builder.WithPredicates(predicate.Funcs{
				DeleteFunc: func(_ event.DeleteEvent) bool {
					return false
				},
			}),
		).
		Complete(r); err != nil {
		return fmt.Errorf("failed to setup NexusRepository reconciler: %w", err)
	}

	return nil
}

// mapSecretToNexusRepository returns reconcile requests for NexusRepositories
// whose HTTP client authentication references the given Secret.
func (r *NexusRepositoryReconciler) mapSecretToNexusRepository(ctx context.Context, secret client.Object) []reconcile.Request {
	repositories := &nexusApi.NexusRepositoryList{}
	if err := r.client.List(ctx, repositories, client.InNamespace(secret.GetNamespace())); err != nil {
		ctrlLog.WithName("secrets_watcher").Error(err, "failed to get NexusRepository list", "secret", secret.GetName())

		return nil
	}

	requests := make([]reconcile.Request, 0, 1)

	for i := range repositories.Items {
		if referencesSecret(&repositories.Items[i].Spec, secret.GetName()) {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&repositories.Items[i])})
		}
	}

	return requests
}

func referencesSecret(spec *nexusApi.NexusRepositorySpec, secretName string) bool {
	repoData, err := nexus.GetRepoData(spec)
	if err != nil {
		return false
	}

	auth := nexus.GetHTTPAuth(repoData.Data)
	if auth == nil {
		return false
	}

	usernameRef, passwordRef := auth.SecretKeyRefs()

	return (usernameRef != nil && usernameRef.Name == secretName) ||
		(passwordRef != nil && passwordRef.Name == secretName)
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
