package user

import (
	"context"
	"fmt"
	"reflect"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

// Reconcile is a reconciler for NexusUser CR.
type Reconcile struct {
	k8sClient client.Client
	scheme    *runtime.Scheme
}

// NewReconcile returns a new instance of Reconcile.
func NewReconcile(k8sClient client.Client, scheme *runtime.Scheme) *Reconcile {
	return &Reconcile{
		k8sClient: k8sClient,
		scheme:    scheme,
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.NexusUser{}, builder.WithPredicates(pred)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to build controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*nexusApi.NexusUser)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*nexusApi.NexusUser)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=nexususers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=nexususers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=nexususers/finalizers,verbs=update

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling NexusUser")

	var instance nexusApi.NexusUser
	if err := r.k8sClient.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("instance not found")
			return
		}

		resultErr = fmt.Errorf("failed to get nexus user from k8s: %w", err)

		return
	}

	log.Info("Reconciliation done")

	return
}
