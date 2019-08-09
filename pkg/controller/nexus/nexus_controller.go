package nexus

import (
	"context"
	"fmt"
	logPrint "log"
	edpv1alpha1 "nexus-operator/pkg/apis/edp/v1alpha1"
	"nexus-operator/pkg/service/nexus"
	"nexus-operator/pkg/service/platform"
	"time"

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
	platformService, _ := platform.NewPlatformService(scheme)

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

	// Watch for changes to primary resource Nexus
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.Nexus{}}, &handler.EnqueueRequestForObject{})
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
	reqLogger.Info("Reconciling Nexus")

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
		logPrint.Printf("[INFO] Installation of %v/%v object with name has been started", instance.Namespace, instance.Name)
		err = r.updateStatus(instance, StatusInstall)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.service.Install(*instance)
	if err != nil {
		logPrint.Printf("[ERROR] Installation of %v/%v object has been failed", instance.Namespace, instance.Name)
		r.updateStatus(instance, StatusFailed)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Status.Status == StatusInstall {
		logPrint.Printf("[INFO] Installation of %v/%v object with name has been finished", instance.Namespace, instance.Name)
		r.updateStatus(instance, StatusCreated)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if dcIsReady, err := r.service.IsDeploymentConfigReady(*instance); err != nil {
		logPrint.Printf("[ERROR] Checking if Deployment config for %v/%v object is ready has been failed", instance.Namespace, instance.Name)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	} else if !dcIsReady {
		logPrint.Printf("[WARNING] Deployment config for %v/%v object is not ready for configuration yet", instance.Namespace, instance.Name)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		logPrint.Printf("[INFO] Configuration of %v/%v object has been started", instance.Namespace, instance.Name)
		err := r.updateStatus(instance, StatusConfiguring)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, isFinished, err := r.service.Configure(*instance)
	if err != nil {
		logPrint.Printf("[ERROR] Configuration of %v/%v object has been failed", instance.Namespace, instance.Name)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	} else if !isFinished {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		logPrint.Printf("[INFO] Configuration of %v/%v object has been finished", instance.Namespace, instance.Name)
		err = r.updateStatus(instance, StatusReady)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	err = r.updateAvailableStatus(instance, true)
	if err != nil {
		logPrint.Printf("[WARNING] Failed update avalability status for Nexus object with name %s. Error - %v", instance.Name, err)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	reqLogger.Info(fmt.Sprintf("Reconciling Nexus component %v/%v has been finished", request.Namespace, request.Name))
	return reconcile.Result{}, nil
}

func (r *ReconcileNexus) updateStatus(instance *edpv1alpha1.Nexus, status string) error {
	instance.Status.Status = status
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			return err
		}
	}

	logPrint.Printf("[INFO] Status for Nexus component %v has been updated to '%v'", instance.Name, status)
	return nil
}

func (r ReconcileNexus) updateAvailableStatus(instance *edpv1alpha1.Nexus, value bool) error {
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return err
			}
		}
		logPrint.Printf("[INFO] Availability status for Nexus component %v has been updated to '%v'", instance.Name, value)
	}

	return nil
}
