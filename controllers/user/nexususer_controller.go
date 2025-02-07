package user

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/controllers"
	"github.com/epam/edp-nexus-operator/controllers/user/chain"
)

// ctrlLog instance ca be used when we can't get logger from context.
var ctrlLog = ctrl.Log.WithName("nexus_user_ctrl")

// NexusUserReconciler reconciles a NexusUser object.
type NexusUserReconciler struct {
	client            client.Client
	scheme            *runtime.Scheme
	apiClientProvider controllers.ApiClientProvider
}

func NewNexusUserReconciler(k8sClient client.Client, scheme *runtime.Scheme, apiClientProvider controllers.ApiClientProvider) *NexusUserReconciler {
	return &NexusUserReconciler{client: k8sClient, scheme: scheme, apiClientProvider: apiClientProvider}
}

//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexususers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexususers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexususers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NexusUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling NexusUser")

	user := &nexusApi.NexusUser{}
	if err := r.client.Get(ctx, req.NamespacedName, user); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get NexusUser: %w", err)
	}

	nexusApiClient, err := r.apiClientProvider.GetNexusApiClientFromNexusRef(ctx, user.Namespace, user)
	if err != nil {
		log.Error(err, "An error has occurred while getting nexus api client")

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	if user.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(user, controllers.NexusOperatorFinalizer) {
			log.Info("Deleting NexusUser")

			if err = chain.NewRemoveUser(nexusApiClient.Security.User).ServeRequest(ctx, user); err != nil {
				log.Error(err, "An error has occurred while deleting NexusUser")

				return ctrl.Result{
					RequeueAfter: controllers.ErrorRequeueTime,
				}, nil
			}

			controllerutil.RemoveFinalizer(user, controllers.NexusOperatorFinalizer)

			if err = r.client.Update(ctx, user); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update NexusUser: %w", err)
			}
		}

		log.Info("NexusUser has been deleted")

		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(user, controllers.NexusOperatorFinalizer) {
		err = r.client.Update(ctx, user)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update NexusUser: %w", err)
		}
	}

	oldStatus := user.Status

	if err = chain.NewCreateUser(nexusApiClient.Security.User, r.client).ServeRequest(ctx, user); err != nil {
		log.Error(err, "An error has occurred while handling NexusUser")

		user.Status.Value = common.StatusError
		user.Status.Error = err.Error()

		if err = r.updateNexusUserStatus(ctx, user, oldStatus); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: controllers.ErrorRequeueTime,
		}, nil
	}

	user.Status.Value = common.StatusCreated
	user.Status.Error = ""

	if err = r.updateNexusUserStatus(ctx, user, oldStatus); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciling NexusUser has been finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NexusUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.NexusUser{}).
		Watches(
			// Watch for changes to Secret resources for updating user related password.
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToNexusUser),
			builder.WithPredicates(predicate.Funcs{
				// We don't need to handle delete event for updating user password.
				DeleteFunc: func(_ event.DeleteEvent) bool {
					return false
				},
			}),
		).
		Complete(r)

	if err != nil {
		return fmt.Errorf("failed to create user controller: %w", err)
	}

	return nil
}

func (r *NexusUserReconciler) updateNexusUserStatus(
	ctx context.Context,
	user *nexusApi.NexusUser,
	oldStatus nexusApi.NexusUserStatus,
) error {
	if user.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update NexusUser status: %w", err)
	}

	return nil
}

// mapSecretToNexusUser returns a list of NexusUser requests to be processed for a given secret.
func (r *NexusUserReconciler) mapSecretToNexusUser(secret client.Object) []reconcile.Request {
	l := ctrlLog.WithName("secrets_watcher").WithValues("secret", secret.GetName())
	requests := make([]reconcile.Request, 0, 1)
	userList := &nexusApi.NexusUserList{}

	err := r.client.List(context.Background(), userList, client.InNamespace(secret.GetNamespace()))
	if err != nil {
		l.Error(err, "failed to get NexusUser list")

		return nil
	}

	for i := range userList.Items {
		name, _, err := chain.ParseSecretRef(userList.Items[i].Spec.Secret)
		if err != nil {
			l.Error(
				err,
				"failed to parse secret ref %q, NexusUser %s",
				userList.Items[i].Spec.Secret,
				userList.Items[i].Name,
			)

			continue
		}

		if name == secret.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKey{
				Name:      userList.Items[i].Name,
				Namespace: userList.Items[i].Namespace,
			}})
		}
	}

	return requests
}
