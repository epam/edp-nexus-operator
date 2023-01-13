package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	commonmock "github.com/epam/edp-common/pkg/mock"

	nexusApi "github.com/epam/edp-nexus-operator/v2/api/edp/v1"
	nexusClient "github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/nexus"
)

type ControllerTestSuite struct {
	suite.Suite
	nx         *nexusApi.Nexus
	nxUser     *nexusApi.NexusUser
	logger     logr.Logger
	clientMock *nexusClient.Mock
	rec        *Reconcile
	scheme     *runtime.Scheme
	clientUser *nexusClient.User
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}

func (s *ControllerTestSuite) SetupTest() {
	s.nx = &nexusApi.Nexus{ObjectMeta: metav1.ObjectMeta{Name: "nx1", Namespace: "ns1"}}
	s.nxUser = &nexusApi.NexusUser{ObjectMeta: metav1.ObjectMeta{Namespace: s.nx.Namespace, Name: "user1"},
		Spec: nexusApi.NexusUserSpec{OwnerName: s.nx.Name, Email: "mktest@example.com"}}
	s.clientUser = instanceSpecToUser(&s.nxUser.Spec)

	s.scheme = runtime.NewScheme()
	err := nexusApi.AddToScheme(s.scheme)
	assert.NoError(s.T(), err)

	s.logger = commonmock.NewLogr()
	fakeK8sClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.nx, s.nxUser).Build()

	s.clientMock = &nexusClient.Mock{}

	rec, err := NewReconcile(fakeK8sClient, s.scheme, s.logger, "kubernetes")
	assert.NoError(s.T(), err)
	s.rec = rec

	rec.getNexusClient = func(ctx context.Context, child nexus.Child) (NexusClient, error) {
		return s.clientMock, nil
	}
}

func (s *ControllerTestSuite) TearDownTest() {
	s.clientMock.AssertExpectations(s.T())
}

func (s *ControllerTestSuite) TestReconcile_Reconcile_Create() {
	s.clientMock.On("GetUser", s.clientUser.Email).Return(nil,
		nexusClient.ErrNotFound("not found"))
	s.clientMock.On("CreateUser", s.clientUser).Return(nil)
	_, err := s.rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name: s.nxUser.Name, Namespace: s.nxUser.Namespace,
	}})

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.NoError(s.T(), err)
	assert.NoError(s.T(), loggerSink.LastError())
}

func (s *ControllerTestSuite) TestReconcile_Reconcile_Update() {
	s.nxUser.Status.ID = "id1"

	s.clientUser.ID = s.nxUser.Status.ID
	fakeK8sClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.nx, s.nxUser).Build()
	s.rec.k8sClient = fakeK8sClient
	s.clientUser.Source = "default"

	s.clientMock.On("UpdateUser", s.clientUser).Return(nil)

	_, err := s.rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name: s.nxUser.Name, Namespace: s.nxUser.Namespace,
	}})
	t := s.T()

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.NoError(t, err)
	assert.NoError(s.T(), loggerSink.LastError())
}

func (s *ControllerTestSuite) TestReconcile_Reconcile_Update_Failure() {
	s.clientMock.On("GetUser", s.clientUser.Email).Return(s.clientUser,
		nil)

	specDuplicate := instanceSpecToUser(&s.nxUser.Spec)
	specDuplicate.Source = "default"

	s.clientMock.On("UpdateUser", specDuplicate).Return(errors.New("update fatal"))

	_, err := s.rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name: s.nxUser.Name, Namespace: s.nxUser.Namespace,
	}})
	t := s.T()

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.NoError(t, err)
	assert.Error(t, loggerSink.LastError())
	assert.Contains(t, loggerSink.LastError().Error(), "update fatal")
}

func (s *ControllerTestSuite) TestReconcile_Reconcile_Delete() {
	s.nxUser.Status.ID = "id1"

	s.nxUser.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	s.nxUser.Finalizers = []string{finalizer}

	s.clientUser.ID = s.nxUser.Status.ID
	s.clientUser.Source = "default"
	fakeK8sClient := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.nx, s.nxUser).Build()
	s.rec.k8sClient = fakeK8sClient

	s.clientMock.On("UpdateUser", s.clientUser).Return(nil)
	s.clientMock.On("DeleteUser", "id1").Return(nil)

	_, err := s.rec.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Name: s.nxUser.Name, Namespace: s.nxUser.Namespace,
	}})

	t := s.T()

	loggerSink, ok := s.logger.GetSink().(*commonmock.Logger)
	assert.True(s.T(), ok)

	assert.NoError(t, err)
	assert.NoError(t, loggerSink.LastError())
}
