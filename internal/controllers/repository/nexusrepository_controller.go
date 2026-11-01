package repository

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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

// passwordRefIndex indexes NexusRepository objects by the name of the Secret in httpClient.authentication.passwordRef.
const passwordRefIndex = "spec.httpClient.authentication.passwordRef.name"

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

// +kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexusrepositories,verbs=get;list;watch;update;patch
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
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&nexusApi.NexusRepository{},
		passwordRefIndex,
		indexPasswordRef,
	); err != nil {
		return fmt.Errorf("failed to index NexusRepository by %s: %w", passwordRefIndex, err)
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.NexusRepository{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToRepositories),
			builder.WithPredicates(secretDataChangedPredicate()),
		).
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

// indexPasswordRef returns the name of the Secret referenced by passwordRef.
func indexPasswordRef(obj client.Object) []string {
	repository, ok := obj.(*nexusApi.NexusRepository)
	if !ok {
		return nil
	}

	repoData, err := nexus.GetRepoData(&repository.Spec)
	if err != nil {
		return nil
	}

	auth := repoData.Authentication()
	if auth == nil || auth.PasswordRef == nil {
		return nil
	}

	return []string{auth.PasswordRef.Name}
}

// mapSecretToRepositories enqueues the NexusRepository objects whose passwordRef names the Secret.
func (r *NexusRepositoryReconciler) mapSecretToRepositories(ctx context.Context, secret client.Object) []reconcile.Request {
	repositories := &nexusApi.NexusRepositoryList{}

	if err := r.client.List(
		ctx,
		repositories,
		client.InNamespace(secret.GetNamespace()),
		client.MatchingFields{passwordRefIndex: secret.GetName()},
	); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "Unable to list NexusRepository objects", "secret", secret.GetName())

		return nil
	}

	requests := make([]reconcile.Request, 0, len(repositories.Items))
	for i := range repositories.Items {
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&repositories.Items[i])})
	}

	return requests
}

// secretDataChangedPredicate passes create, delete and generic events, and update events that change Secret data.
func secretDataChangedPredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, okOld := e.ObjectOld.(*corev1.Secret)
			newSecret, okNew := e.ObjectNew.(*corev1.Secret)

			return okOld && okNew && !equality.Semantic.DeepEqual(oldSecret.Data, newSecret.Data)
		},
	}
}
