package nexus

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nexusApi "github.com/epam/edp-nexus-operator/v2/api/v1"
	"github.com/epam/edp-nexus-operator/v2/controllers/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/nexus"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform"
)

const (
	StatusInstall          = "installing"
	StatusFailed           = "failed"
	StatusCreated          = "created"
	StatusConfiguring      = "configuring"
	StatusConfigured       = "configured"
	StatusExposeStart      = "exposing config"
	StatusExposeFinish     = "config exposed"
	StatusIntegrationStart = "integration started"
	StatusReady            = "ready"
	RequeueAfterSeconds10  = 10 * time.Second
	RequeueAfterSeconds30  = 30 * time.Second
	RequeueAfterSeconds60  = 60 * time.Second
)

func NewReconcileNexus(c client.Client, scheme *runtime.Scheme, log logr.Logger) (*ReconcileNexus, error) {
	ps, err := platform.NewPlatformService(helper.GetPlatformTypeEnv(), scheme, c)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform service: %w", err)
	}

	return &ReconcileNexus{
		client:  c,
		scheme:  scheme,
		service: nexus.NewService(ps, c, scheme),
		log:     log.WithName("nexus"),
	}, nil
}

type ReconcileNexus struct {
	client  client.Client
	scheme  *runtime.Scheme
	service nexus.Service
	log     logr.Logger
}

func (r *ReconcileNexus) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*nexusApi.Nexus)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*nexusApi.Nexus)
			if !ok {
				return false
			}

			return oldObject.Status == newObject.Status
		},
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&nexusApi.Nexus{}, builder.WithPredicates(p)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create cpntroller manager: %w", err)
	}

	return nil
}

func (r *ReconcileNexus) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling has been started")

	instance := &nexusApi.Nexus{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get Nexus instance from k8s: %w", err)
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		log.Info("Installation has been started")

		if err := r.updateStatus(ctx, instance, StatusInstall); err != nil {
			return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, err
		}
	}

	if instance.Status.Status == StatusInstall {
		log.Info("Installation has been finished")

		if err := r.updateStatus(ctx, instance, StatusCreated); err != nil {
			return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, err
		}
	}

	if ready, err := r.service.IsDeploymentReady(instance); err != nil {
		return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, fmt.Errorf("failed to check if deployment config is ready: %w", err)
	} else if !*ready {
		log.Info("Deployment config is not ready for configuration yet")
		return reconcile.Result{RequeueAfter: RequeueAfterSeconds60}, nil
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		log.Info("Configuration has started")

		if err := r.updateStatus(ctx, instance, StatusConfiguring); err != nil {
			return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, err
		}
	}

	instance, isFinished, err := r.service.Configure(instance)
	if err != nil {
		log.Error(err, "Configuration has failed")
		return reconcile.Result{RequeueAfter: RequeueAfterSeconds30}, fmt.Errorf("service configuration failed : %w", err)
	}

	if !isFinished {
		return reconcile.Result{RequeueAfter: RequeueAfterSeconds30}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		log.Info("Configuration has finished")

		if updErr := r.updateStatus(ctx, instance, StatusConfigured); updErr != nil {
			return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, updErr
		}
	}

	if instance.Status.Status == StatusConfigured {
		log.Info("Exposing configuration has started")

		if updErr := r.updateStatus(ctx, instance, StatusExposeStart); updErr != nil {
			return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, updErr
		}
	}

	instance, err = r.service.ExposeConfiguration(ctx, instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, fmt.Errorf("failed to expose configuration: %w", err)
	}

	if instance.Status.Status == StatusExposeStart {
		log.Info("Exposing configuration has finished")

		if updErr := r.updateStatus(ctx, instance, StatusExposeFinish); updErr != nil {
			return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, updErr
		}
	}

	if instance.Status.Status == StatusExposeFinish {
		log.Info("Exposing configuration has started")

		if updErr := r.updateStatus(ctx, instance, StatusIntegrationStart); updErr != nil {
			return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, updErr
		}
	}

	instance, err = r.service.Integration(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, fmt.Errorf("integration failed: %w", err)
	}

	if instance.Status.Status == StatusIntegrationStart {
		log.Info("Exposing configuration has started")

		if updErr := r.updateStatus(ctx, instance, StatusReady); updErr != nil {
			return reconcile.Result{RequeueAfter: RequeueAfterSeconds10}, updErr
		}
	}

	if err = r.updateAvailableStatus(ctx, instance, true); err != nil {
		log.Info("Failed to update availability status")

		return reconcile.Result{RequeueAfter: RequeueAfterSeconds30}, err
	}

	log.Info("Reconciling has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileNexus) updateStatus(ctx context.Context, instance *nexusApi.Nexus, newStatus string) error {
	log := r.log.
		WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).
		WithName("status_update")

	currentStatus := instance.Status.Status
	instance.Status.Status = newStatus
	instance.Status.LastTimeUpdated = metav1.NewTime(time.Now())

	if err := r.client.Status().Update(ctx, instance); err != nil {
		if updErr := r.client.Update(ctx, instance); updErr != nil {
			return fmt.Errorf("failed to update status from '%v' to '%v': %w", currentStatus, newStatus, updErr)
		}
	}

	log.Info(fmt.Sprintf("Status has been updated to '%v'", newStatus))

	return nil
}

func (r *ReconcileNexus) updateAvailableStatus(ctx context.Context, instance *nexusApi.Nexus, value bool) error {
	log := r.log.
		WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).
		WithName("status_update")

	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = metav1.NewTime(time.Now())

		if err := r.client.Status().Update(ctx, instance); err != nil {
			if updErr := r.client.Update(ctx, instance); updErr != nil {
				return fmt.Errorf("failed to update availability status to %v: %w", value, updErr)
			}
		}

		log.Info(fmt.Sprintf("Availability status has been updated to '%v'", value))
	}

	return nil
}
