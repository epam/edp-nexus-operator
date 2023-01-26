package user

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dchest/uniuri"
	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nexusApi "github.com/epam/edp-nexus-operator/v2/api/v1"
	"github.com/epam/edp-nexus-operator/v2/controllers/helper"
	nexusClient "github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/nexus"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform"
)

const (
	finalizer = "nexus.user.operator"
	crDefault = "default"
)

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
		return nil, fmt.Errorf("failed to create platform service: %w", err)
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
	nc, err := r.service.ClientForNexusChild(ctx, child)
	if err != nil {
		return nc, fmt.Errorf("failed to get client for nexus chield: %w", err)
	}

	return nc, nil
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
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling Nexus User")

	var instance nexusApi.NexusUser
	if err := r.k8sClient.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("instance not found")
			return
		}

		resultErr = fmt.Errorf("failed to get nexus user from k8s: %w", err)

		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()

		if errStatusUpdate := r.k8sClient.Status().Update(ctx, &instance); errStatusUpdate != nil {
			resultErr = errStatusUpdate
		}

		result.RequeueAfter = helper.FailureReconciliationTimeout

		log.Error(err, "an error has occurred while handling nexus user", "name",
			request.Name)
	}

	log.Info("Reconciliation done")

	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *nexusApi.NexusUser) error {
	nxCl, err := r.getNexusClient(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create nexus client for child: %w", err)
	}

	if err = r.syncUser(ctx, instance, nxCl); err != nil {
		return fmt.Errorf("failed to sync user: %w", err)
	}

	if _, err = r.deleteResource(ctx, instance, nxCl); err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	return nil
}

func (*Reconcile) syncUser(ctx context.Context, instance *nexusApi.NexusUser, nxCl NexusClient) error {
	specUsr := instanceSpecToUser(&instance.Spec)

	if instance.Status.ID == "" {
		usr, err := nxCl.GetUser(ctx, specUsr.Email)
		if nexusClient.IsErrNotFound(err) {
			specUsr.Password = uniuri.New()
			if err = nxCl.CreateUser(ctx, specUsr); err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}

			instance.Status.ID = specUsr.ID

			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		instance.Status.ID = usr.ID
	}

	specUsr.ID = instance.Status.ID
	specUsr.Source = crDefault

	if err := nxCl.UpdateUser(ctx, specUsr); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func instanceSpecToUser(spec *nexusApi.NexusUserSpec) *nexusClient.User {
	return &nexusClient.User{
		Roles:     spec.Roles,
		FirstName: spec.FirstName,
		LastName:  spec.LastName,
		Email:     spec.Email,
		Status:    spec.Status,
		ID:        spec.UserID,
	}
}

func (r *Reconcile) deleteResource(ctx context.Context, instance *nexusApi.NexusUser,
	nxCl NexusClient) (bool, error) {
	finalizers := instance.GetFinalizers()

	if instance.GetDeletionTimestamp().IsZero() {
		if !helper.ContainsString(finalizers, finalizer) {
			finalizers = append(finalizers, finalizer)
			instance.SetFinalizers(finalizers)

			if err := r.k8sClient.Update(ctx, instance); err != nil {
				return false, fmt.Errorf("failed to update deletable object: %w", err)
			}
		}

		return false, nil
	}

	if err := nxCl.DeleteUser(ctx, instance.Status.ID); err != nil {
		return false, fmt.Errorf("failed to delete resource: %w", err)
	}

	if helper.ContainsString(finalizers, finalizer) {
		finalizers = helper.RemoveString(finalizers, finalizer)
		instance.SetFinalizers(finalizers)

		if err := r.k8sClient.Update(ctx, instance); err != nil {
			return false, fmt.Errorf("failed to update instance status: %w", err)
		}
	}

	return true, nil
}
