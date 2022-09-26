package nexus

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-nexus-operator/v2/mocks"
	sMock "github.com/epam/edp-nexus-operator/v2/mocks/nexus"
	nexusApi "github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1"
)

const name = "name"
const namespace = "namespace"

func createNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

func createInstanceByStatus(status string) *nexusApi.Nexus {
	return &nexusApi.Nexus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name},
		Status: nexusApi.NexusStatus{Status: status},
	}
}

func createClient(instance *nexusApi.Nexus) k8sClient.Client {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance).Build()
}

func TestReconcileNexus_Reconcile_BadClient(t *testing.T) {
	ctx := context.Background()

	client := fake.NewClientBuilder().Build()
	reconcileNexus := ReconcileNexus{
		client: client,
		log:    logr.DiscardLogger{},
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.True(t, runtime.IsNotRegisteredError(err))
	assert.Equal(t, reconcile.Result{}, result)
}

func TestReconcileNexus_Reconcile_EmptyClient(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconcileNexus := ReconcileNexus{
		client: client,
		log:    logr.DiscardLogger{},
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestReconcileNexus_Reconcile_UpdateStatusFailedErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusFailed)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)

	reconcileNexus := ReconcileNexus{
		client: &clientMock,
		log:    logr.DiscardLogger{},
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update status from"))
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_UpdateStatusInstallErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusInstall)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)

	reconcileNexus := ReconcileNexus{
		client: &clientMock,
		log:    logr.DiscardLogger{},
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update status from"))
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_IsDeploymentReadyErr(t *testing.T) {
	ctx := context.Background()

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusInstall)
	client := createClient(instance)
	service := sMock.Service{}
	service.On("IsDeploymentReady").Return(nil, errTest)

	reconcileNexus := ReconcileNexus{
		client:  client,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Checking if Deployment config is ready has been failed"))
	service.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_IsDeploymentReadyFalse(t *testing.T) {
	ctx := context.Background()

	instance := createInstanceByStatus(StatusInstall)
	client := createClient(instance)
	service := sMock.Service{}
	ok := false
	service.On("IsDeploymentReady").Return(&ok, nil)

	reconcileNexus := ReconcileNexus{
		client:  client,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 60 * time.Second}, result)
	assert.NoError(t, err)
	service.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_UpdateStatusCreatedErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()
	service := sMock.Service{}
	ok := true

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusCreated)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)
	service.On("IsDeploymentReady").Return(&ok, nil)

	reconcileNexus := ReconcileNexus{
		client:  &clientMock,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update status from"))
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_ConfigureErr(t *testing.T) {
	ctx := context.Background()
	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusInstall)
	client := createClient(instance)
	service := sMock.Service{}
	ok := true
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, false, errTest)

	reconcileNexus := ReconcileNexus{
		client:  client,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Configuration failed"))
	service.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_ConfigureFalse(t *testing.T) {
	ctx := context.Background()

	instance := createInstanceByStatus(StatusInstall)
	client := createClient(instance)
	service := sMock.Service{}
	ok := true
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, false, nil)

	reconcileNexus := ReconcileNexus{
		client:  client,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Second}, result)
	assert.NoError(t, err)
	service.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_UpdateStatusConfiguringErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()
	service := sMock.Service{}
	ok := true

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusConfiguring)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)

	reconcileNexus := ReconcileNexus{
		client:  &clientMock,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update status from"))
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_UpdateStatusConfiguredErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()
	service := sMock.Service{}
	ok := true

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusConfigured)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)

	reconcileNexus := ReconcileNexus{
		client:  &clientMock,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update status from"))
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_ExposeConfigurationErr(t *testing.T) {
	ctx := context.Background()
	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusInstall)
	client := createClient(instance)
	service := sMock.Service{}
	ok := true
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)
	service.On("ExposeConfiguration", ctx, *instance).Return(instance, errTest)

	reconcileNexus := ReconcileNexus{
		client:  client,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Exposing configuration failed"))
	service.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_UpdateStatusExposeStartErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()
	service := sMock.Service{}
	ok := true

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusExposeStart)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)
	service.On("ExposeConfiguration", ctx, *instance).Return(instance, nil)

	reconcileNexus := ReconcileNexus{
		client:  &clientMock,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update status from"))
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_UpdateStatusExposeFinishErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()
	service := sMock.Service{}
	ok := true

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusExposeFinish)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)
	service.On("ExposeConfiguration", ctx, *instance).Return(instance, nil)

	reconcileNexus := ReconcileNexus{
		client:  &clientMock,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update status from"))
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_IntegrationErr(t *testing.T) {
	ctx := context.Background()
	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusInstall)
	client := createClient(instance)
	service := sMock.Service{}
	ok := true
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)
	service.On("ExposeConfiguration", ctx, *instance).Return(instance, nil)
	service.On("Integration", *instance).Return(instance, errTest)

	reconcileNexus := ReconcileNexus{
		client:  client,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Integration failed"))
	service.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_UpdateStatusIntegrationStartErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()
	service := sMock.Service{}
	ok := true

	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusIntegrationStart)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)
	service.On("ExposeConfiguration", ctx, *instance).Return(instance, nil)
	service.On("Integration", *instance).Return(instance, nil)

	reconcileNexus := ReconcileNexus{
		client:  &clientMock,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update status from"))
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile_UpdateStatusReadyErr(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()
	service := sMock.Service{}
	ok := true
	errTest := errors.New("test")

	instance := createInstanceByStatus(StatusReady)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	clientMock.On("Update").Return(errTest)
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)
	service.On("ExposeConfiguration", ctx, *instance).Return(instance, nil)
	service.On("Integration", *instance).Return(instance, nil)

	reconcileNexus := ReconcileNexus{
		client:  &clientMock,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Second}, result)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't update availability status"))
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
}

func TestReconcileNexus_Reconcile(t *testing.T) {
	statusWriter := &mocks.StatusWriter{}
	clientMock := mocks.Client{}
	ctx := context.Background()
	service := sMock.Service{}
	ok := true

	instance := createInstanceByStatus(StatusReady)
	client := createClient(instance)

	clientMock.On("Get", createNamespacedName(), &nexusApi.Nexus{}).Return(client)
	clientMock.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(nil)
	service.On("IsDeploymentReady").Return(&ok, nil)
	service.On("Configure").Return(instance, true, nil)
	service.On("ExposeConfiguration", ctx, *instance).Return(instance, nil)
	service.On("Integration", *instance).Return(instance, nil)

	reconcileNexus := ReconcileNexus{
		client:  &clientMock,
		log:     logr.DiscardLogger{},
		service: &service,
	}

	req := reconcile.Request{
		NamespacedName: createNamespacedName(),
	}

	result, err := reconcileNexus.Reconcile(ctx, req)
	assert.Equal(t, reconcile.Result{}, result)
	assert.NoError(t, err)
	clientMock.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
	service.AssertExpectations(t)
}
