package nexus

import (
	"context"
	"fmt"
	"time"

	"github.com/datadrivers/go-nexus-client/nexus3"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/controllers/nexus/chain"
)

const (
	defaultRequeueTime = time.Second * 30
	successRequeueTime = time.Minute * 10
)

type apiClientProvider interface {
	GetNexusApiClientFromNexus(ctx context.Context, nexus *nexusApi.Nexus) (*nexus3.NexusClient, error)
}

func NewNexusReconciler(c client.Client, scheme *runtime.Scheme, nexusApiProvider apiClientProvider) *NexusReconciler {
	return &NexusReconciler{
		client:            c,
		scheme:            scheme,
		apiClientProvider: nexusApiProvider,
	}
}

type NexusReconciler struct {
	client            client.Client
	scheme            *runtime.Scheme
	apiClientProvider apiClientProvider
}

//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexuses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=edp.epam.com,namespace=placeholder,resources=nexuses/finalizers,verbs=update
//+kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NexusReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling Nexus")

	nexus := &nexusApi.Nexus{}
	if err := r.client.Get(ctx, req.NamespacedName, nexus); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get Nexus instance from k8s: %w", err)
	}

	oldStatus := nexus.Status

	nexusApiClient, err := r.apiClientProvider.GetNexusApiClientFromNexus(ctx, nexus)
	if err != nil {
		nexus.Status.Error = err.Error()
		nexus.Status.Connected = false

		if statusErr := r.updateNexusStatus(ctx, nexus, oldStatus); statusErr != nil {
			return reconcile.Result{}, statusErr
		}

		return reconcile.Result{RequeueAfter: defaultRequeueTime}, fmt.Errorf("failed to get nexus api client: %w", err)
	}

	if err = chain.NewCheckConnection(nexusApiClient.Security.User).ServeRequest(ctx, nexus); err != nil {
		nexus.Status.Error = err.Error()
		nexus.Status.Connected = false

		if statusErr := r.updateNexusStatus(ctx, nexus, oldStatus); statusErr != nil {
			return reconcile.Result{}, statusErr
		}

		return reconcile.Result{RequeueAfter: defaultRequeueTime}, fmt.Errorf("failed to serve request: %w", err)
	}

	nexus.Status.Connected = true
	nexus.Status.Error = ""

	if err = r.updateNexusStatus(ctx, nexus, oldStatus); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Reconciling Nexus is finished")

	return reconcile.Result{
		RequeueAfter: successRequeueTime,
	}, nil
}

func (r *NexusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.Nexus{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create controller manager: %w", err)
	}

	return nil
}

func (r *NexusReconciler) updateNexusStatus(ctx context.Context, nexus *nexusApi.Nexus, oldStatus nexusApi.NexusStatus) error {
	if nexus.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, nexus); err != nil {
		return fmt.Errorf("failed to update Nexus status: %w", err)
	}

	return nil
}
