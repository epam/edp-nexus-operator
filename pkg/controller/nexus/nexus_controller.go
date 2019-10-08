package nexus

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/nexus-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/nexus-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/nexus-operator/v2/pkg/service/nexus"
	"github.com/epmd-edp/nexus-operator/v2/pkg/service/platform"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	errorsf "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
)

var log = logf.Log.WithName("controller_nexus")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Nexus Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	client := mgr.GetClient()
	platformType := helper.GetPlatformTypeEnv()
	platformService, err := platform.NewPlatformService(platformType,scheme, &client)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	nexusService := nexus.NewNexusService(platformService, client)

	return &ReconcileNexus{
		client:  client,
		scheme:  scheme,
		service: nexusService,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nexus-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*edpv1alpha1.Nexus)
			newObject := e.ObjectNew.(*edpv1alpha1.Nexus)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}

	// Watch for changes to primary resource Nexus
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.Nexus{}}, &handler.EnqueueRequestForObject{}, p)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNexus{}

// ReconcileNexus reconciles a Nexus object
type ReconcileNexus struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	service nexus.NexusService
}

// Reconcile reads that state of the cluster for a Nexus object and makes changes based on the state read
// and what is in the Nexus.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNexus) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling has been started")
	// Fetch the Nexus instance
	instance := &edpv1alpha1.Nexus{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		reqLogger.Info("Installation has been started")
		err = r.updateStatus(instance, StatusInstall)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	instance, err = r.service.Install(*instance)
	if err != nil {
		r.updateStatus(instance, StatusFailed)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, errorsf.Wrap(err, "Installation has been failed")
	}

	if instance.Status.Status == StatusInstall {
		reqLogger.Info("Installation has been finished")
		r.updateStatus(instance, StatusCreated)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	}

	if ready, err := r.service.IsDeploymentReady(*instance); err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, errorsf.Wrap(err, "Checking if Deployment config is ready has been failed")
	} else if !*ready {
		reqLogger.Info("Deployment config is not ready for configuration yet")
		return reconcile.Result{RequeueAfter: 60 * time.Second}, nil
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		reqLogger.Info("Configuration has started")
		err := r.updateStatus(instance, StatusConfiguring)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	instance, isFinished, err := r.service.Configure(*instance)
	if err != nil {
		reqLogger.Error(err, "Configuration has failed")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, errorsf.Wrap(err, "Configuration failed")
	} else if !isFinished {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		reqLogger.Info("Configuration has finished")
		err = r.updateStatus(instance, StatusConfigured)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	if instance.Status.Status == StatusConfigured {
		reqLogger.Info("Exposing configuration has started")
		err = r.updateStatus(instance, StatusExposeStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	instance, err = r.service.ExposeConfiguration(*instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, errorsf.Wrap(err, "Exposing configuration failed")
	}

	if instance.Status.Status == StatusExposeStart {
		reqLogger.Info("Exposing configuration has finished")
		err = r.updateStatus(instance, StatusExposeFinish)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	if instance.Status.Status == StatusExposeFinish {
		reqLogger.Info("Exposing configuration has started")
		err = r.updateStatus(instance, StatusIntegrationStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	instance, err = r.service.Integration(*instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, errorsf.Wrap(err, "Integration failed")
	}

	if instance.Status.Status == StatusIntegrationStart {
		reqLogger.Info("Exposing configuration has started")
		err = r.updateStatus(instance, StatusReady)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	err = r.updateAvailableStatus(instance, true)
	if err != nil {
		reqLogger.Info("Failed to update availability status")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, err
	}

	reqLogger.Info("Reconciling has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileNexus) updateStatus(instance *edpv1alpha1.Nexus, newStatus string) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	currentStatus := instance.Status.Status
	instance.Status.Status = newStatus
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			return errorsf.Wrapf(err, "couldn't update status from '%v' to '%v'", currentStatus, newStatus)
		}
	}
	reqLogger.Info(fmt.Sprintf("Status has been updated to '%v'", newStatus))
	return nil
}

func (r ReconcileNexus) updateAvailableStatus(instance *edpv1alpha1.Nexus, value bool) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return errorsf.Wrapf(err, "couldn't update availability status to %v", value)
			}
		}
		reqLogger.Info(fmt.Sprintf("Availability status has been updated to '%v'", value))
	}
	return nil
}
