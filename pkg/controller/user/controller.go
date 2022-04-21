package user

import (
	"context"
	"reflect"

	"github.com/dchest/uniuri"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1alpha1"
	nexusClient "github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/nexus"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform"
)

const finalizer = "nexus.user.operator"

type Reconcile struct {
	k8sClient      client.Client
	service        nexus.Service
	log            logr.Logger
	getNexusClient func(ctx context.Context, child nexus.Child) (NexusClient, error)
}

type NexusClient interface {
	CreateUser(ctx context.Context, u *nexusClient.User) error
	UpdateUser(ctx context.Context, u *nexusClient.User) error
	DeleteUser(ctx context.Context, ID string) error
	GetUser(ctx context.Context, email string) (*nexusClient.User, error)
}

func NewReconcile(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger, platformType string) (*Reconcile, error) {
	ps, err := platform.NewPlatformService(platformType, scheme, k8sClient)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	r := Reconcile{
		service:   nexus.NewService(ps, k8sClient, scheme),
		log:       log,
		k8sClient: k8sClient,
	}

	r.getNexusClient = r.clientForNexusChild
	return &r, nil
}

func (r *Reconcile) clientForNexusChild(ctx context.Context, child nexus.Child) (NexusClient, error) {
	return r.service.ClientForNexusChild(ctx, child)
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.NexusUser{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*v1alpha1.NexusUser)
	no := e.ObjectNew.(*v1alpha1.NexusUser)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling Nexus User")

	var instance v1alpha1.NexusUser
	if err := r.k8sClient.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("instance not found")
			return
		}

		resultErr = errors.Wrap(err, "unable to get nexus user from k8s")
		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = helper.FailureReconciliationTimeout
		log.Error(err, "an error has occurred while handling nexus user", "name",
			request.Name)
	} else {
		instance.Status.Value = helper.StatusOK
	}

	if err := r.k8sClient.Status().Update(ctx, &instance); err != nil {
		resultErr = err
	}

	log.Info("Reconciliation done")

	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *v1alpha1.NexusUser) error {
	nxCl, err := r.getNexusClient(ctx, instance)
	if err != nil {
		return errors.Wrap(err, "unable to create nexus client for child")
	}

	if err := r.syncUser(ctx, instance, nxCl); err != nil {
		return errors.Wrap(err, "unable to sync user")
	}

	if _, err := r.deleteResource(ctx, instance, nxCl); err != nil {
		return errors.Wrap(err, "unable to delete resource")
	}

	return nil
}

func (r *Reconcile) syncUser(ctx context.Context, instance *v1alpha1.NexusUser, nxCl NexusClient) error {
	specUsr := instanceSpecToUser(&instance.Spec)

	if instance.Status.ID == "" {
		usr, err := nxCl.GetUser(ctx, specUsr.Email)
		if nexusClient.IsErrNotFound(err) {
			specUsr.Password = uniuri.New()
			if err := nxCl.CreateUser(ctx, specUsr); err != nil {
				return errors.Wrap(err, "unable to create user")
			}
			instance.Status.ID = specUsr.ID

			return nil
		} else if err != nil {
			return errors.Wrap(err, "unknown error")
		}

		instance.Status.ID = usr.ID
	}

	specUsr.ID = instance.Status.ID
	specUsr.Source = "default"
	if err := nxCl.UpdateUser(ctx, specUsr); err != nil {
		return errors.Wrap(err, "unable to update user")
	}

	return nil
}

func instanceSpecToUser(spec *v1alpha1.NexusUserSpec) *nexusClient.User {
	return &nexusClient.User{
		Roles:     spec.Roles,
		FirstName: spec.FirstName,
		LastName:  spec.LastName,
		Email:     spec.Email,
		Status:    spec.Status,
		ID:        spec.UserID,
	}
}

func (r *Reconcile) deleteResource(ctx context.Context, instance *v1alpha1.NexusUser,
	nxCl NexusClient) (bool, error) {
	finalizers := instance.GetFinalizers()

	if instance.GetDeletionTimestamp().IsZero() {
		if !helper.ContainsString(finalizers, finalizer) {
			finalizers = append(finalizers, finalizer)
			instance.SetFinalizers(finalizers)

			if err := r.k8sClient.Update(ctx, instance); err != nil {
				return false, errors.Wrap(err, "unable to update deletable object")
			}
		}

		return false, nil
	}

	if err := nxCl.DeleteUser(ctx, instance.Status.ID); err != nil {
		return false, errors.Wrap(err, "unable to delete resource")
	}

	if helper.ContainsString(finalizers, finalizer) {
		finalizers = helper.RemoveString(finalizers, finalizer)
		instance.SetFinalizers(finalizers)

		if err := r.k8sClient.Update(ctx, instance); err != nil {
			return false, errors.Wrap(err, "unable to update instance status")
		}
	}

	return true, nil
}
