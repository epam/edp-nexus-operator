package user

import (
	"context"
	"testing"
	"time"

	commonMock "github.com/epam/edp-common/pkg/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1alpha1"
	nexusClient "github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/nexus"
)

var (
	nx     = v1alpha1.Nexus{ObjectMeta: metav1.ObjectMeta{Name: "nx1", Namespace: "ns1"}}
	nxUser = v1alpha1.NexusUser{ObjectMeta: metav1.ObjectMeta{Namespace: nx.Namespace, Name: "user1"},
		Spec: v1alpha1.NexusUserSpec{OwnerName: nx.Name, Email: "mktest@example.com"}}
)

func initController(t *testing.T) (*Reconcile, *commonMock.Logger, *nexusClient.Mock) {
	sch := runtime.NewScheme()
	v1alpha1.RegisterTypes(sch)

	logger := commonMock.Logger{}
	fakeK8sClient := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&nx, &nxUser).Build()

	clientMock := nexusClient.Mock{}

	rec, err := NewReconcile(fakeK8sClient, sch, &logger, "kubernetes")
	if err != nil {
		t.Fatal(err)
	}

	rec.getNexusClient = func(ctx context.Context, child nexus.Child) (NexusClient, error) {
		return &clientMock, nil
	}

	return rec, &logger, &clientMock
}

func TestReconcile_Reconcile_Create(t *testing.T) {
	rec, logger, clientMock := initController(t)

	nexusClientUser := instanceSpecToUser(&nxUser.Spec)
	clientMock.On("CreateUser", nexusClientUser).Return(nil)
	_, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name: nxUser.Name, Namespace: nxUser.Namespace,
	}})

	if err != nil {
		t.Fatal(err)
	}

	if err := logger.LastError(); err != nil {
		t.Fatal(err)
	}
}

func TestReconcile_Reconcile_Update(t *testing.T) {
	nexusClientUser := instanceSpecToUser(&nxUser.Spec)
	nxUser.Status.ID = "id1"
	rec, logger, clientMock := initController(t)

	nexusClientUser.ID = nxUser.Status.ID
	nexusClientUser.Source = "default"

	clientMock.On("UpdateUser", nexusClientUser).Return(nil)

	_, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name: nxUser.Name, Namespace: nxUser.Namespace,
	}})

	if err != nil {
		t.Fatal(err)
	}

	if err := logger.LastError(); err != nil {
		t.Fatal(err)
	}
}

func TestReconcile_Reconcile_Delete(t *testing.T) {
	nexusClientUser := instanceSpecToUser(&nxUser.Spec)
	nxUser.Status.ID = "id1"
	rec, logger, clientMock := initController(t)

	nxUser.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	nexusClientUser.ID = nxUser.Status.ID
	nexusClientUser.Source = "default"

	clientMock.On("UpdateUser", nexusClientUser).Return(nil)
	clientMock.On("DeleteUser", "id1").Return(nil)

	_, err := rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name: nxUser.Name, Namespace: nxUser.Namespace,
	}})

	if err != nil {
		t.Fatal(err)
	}

	if err := logger.LastError(); err != nil {
		t.Fatal(err)
	}
}
