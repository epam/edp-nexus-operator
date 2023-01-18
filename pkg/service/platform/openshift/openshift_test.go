package openshift

import (
	"context"
	"fmt"
	"os"
	"testing"

	appv1 "github.com/openshift/api/apps/v1"
	v1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nexusApi "github.com/epam/edp-nexus-operator/v2/api/edp/v1"
	openshiftMock "github.com/epam/edp-nexus-operator/v2/mocks/openshift"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
)

const (
	name      = "name"
	namespace = "ns"
)

func createDeploymentConfInstance(container []coreV1Api.Container) appv1.DeploymentConfig {
	return appv1.DeploymentConfig{
		Spec: appv1.DeploymentConfigSpec{
			Template: &coreV1Api.PodTemplateSpec{
				Spec: coreV1Api.PodSpec{
					Containers: container,
				},
			},
		},
	}
}

func createContainer(instance *nexusApi.Nexus) coreV1Api.Container {
	return coreV1Api.Container{
		Name:            "keycloak-proxy",
		Image:           instance.Spec.KeycloakSpec.ProxyImage,
		ImagePullPolicy: coreV1Api.PullIfNotPresent,
		Ports: []coreV1Api.ContainerPort{
			{
				ContainerPort: nexusDefaultSpec.NexusKeycloakProxyPort,
				Protocol:      coreV1Api.ProtocolTCP,
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: coreV1Api.TerminationMessageReadFile,
		Args:                     nil,
	}
}

func createObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func TestOpenshiftService_Init(t *testing.T) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	service := OpenshiftService{}

	err = service.Init(restConfig, scheme, client)
	assert.NoError(t, err)
}

func TestOpenshiftService_GetRouteByCr_ListErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}

	routeClient.On("Routes", namespace).Return(routes)
	routes.On("List", context.TODO(), metav1.ListOptions{}).Return(nil, fmt.Errorf("test"))

	service := OpenshiftService{routeClient: &routeClient}

	_, err := service.GetRouteByCr(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve services list from the cluster")
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_GetRouteByCr_NotInList(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	route := v1.Route{}
	list := v1.RouteList{Items: []v1.Route{route}}
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}
	routeClient.On("Routes", namespace).Return(routes)
	routes.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	service := OpenshiftService{routeClient: &routeClient}

	result, err := service.GetRouteByCr(instance)
	assert.NoError(t, err)
	assert.Nil(t, result)
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_GetRouteByCr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	route := v1.Route{ObjectMeta: createObjectMeta()}
	list := v1.RouteList{Items: []v1.Route{route}}
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}

	routeClient.On("Routes", namespace).Return(routes)
	routes.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	service := OpenshiftService{routeClient: &routeClient}

	result, err := service.GetRouteByCr(instance)
	assert.NoError(t, err)
	assert.Equal(t, &route, result)
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_GetExternalUrl_NotFound(t *testing.T) {
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}
	routeClient.On("Routes", namespace).Return(routes)
	routes.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))

	service := OpenshiftService{routeClient: &routeClient}

	// nolint
	_, _, _, err := service.GetExternalUrl(namespace, name)
	assert.NoError(t, err)
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_GetExternalUrl_GetErr(t *testing.T) {
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}
	errTest := fmt.Errorf("test")

	routeClient.On("Routes", namespace).Return(routes)
	routes.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	service := OpenshiftService{routeClient: &routeClient}

	// nolint
	_, _, _, err := service.GetExternalUrl(namespace, name)

	assert.Contains(t, err.Error(), "failed to get route")
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_GetExternalUrl(t *testing.T) {
	route := v1.Route{
		ObjectMeta: createObjectMeta(),
		Spec: v1.RouteSpec{
			Path: "domain",
			TLS: &v1.TLSConfig{
				Termination: "https",
			},
		},
	}
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}

	routeClient.On("Routes", namespace).Return(routes)
	routes.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&route, nil)

	service := OpenshiftService{routeClient: &routeClient}

	url, s, s2, err := service.GetExternalUrl(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, "https://domain", url)
	assert.Equal(t, "", s)
	assert.Equal(t, "https", s2)
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_UpdateExternalTargetPath_GetRouteByCr(t *testing.T) {
	orString := intstr.IntOrString{}
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}

	routeClient.On("Routes", namespace).Return(routes)
	routes.On("List", context.TODO(), metav1.ListOptions{}).Return(nil, fmt.Errorf("test"))

	service := OpenshiftService{routeClient: &routeClient}

	err := service.UpdateExternalTargetPath(instance, orString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get route or error")
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_UpdateExternalTargetPath_AlreadyUpdated(t *testing.T) {
	intOrString := intstr.IntOrString{}
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	route := v1.Route{
		ObjectMeta: createObjectMeta(),
		Spec: v1.RouteSpec{
			Port: &v1.RoutePort{
				TargetPort: intOrString,
			},
		},
	}
	list := v1.RouteList{Items: []v1.Route{route}}
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}

	routeClient.On("Routes", namespace).Return(routes)
	routes.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	service := OpenshiftService{routeClient: &routeClient}

	err := service.UpdateExternalTargetPath(instance, intOrString)
	assert.NoError(t, err)
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_UpdateExternalTargetPath(t *testing.T) {
	intOrString := intstr.IntOrString{}
	intTwo := intstr.IntOrString{
		Type:   0,
		IntVal: 2,
		StrVal: "",
	}
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	route := v1.Route{
		ObjectMeta: createObjectMeta(),
		Spec: v1.RouteSpec{
			Port: &v1.RoutePort{
				TargetPort: intTwo,
			},
		},
	}
	expectedRoute := v1.Route{
		ObjectMeta: createObjectMeta(),
		Spec: v1.RouteSpec{
			Port: &v1.RoutePort{
				TargetPort: intOrString,
			},
		},
	}
	list := v1.RouteList{Items: []v1.Route{route}}
	routeClient := openshiftMock.RouteV1Interface{}
	routes := &openshiftMock.RouteInterface{}

	routeClient.On("Routes", namespace).Return(routes)
	routes.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)
	routes.On("Update", context.TODO(), &expectedRoute, metav1.UpdateOptions{}).Return(nil, nil)

	service := OpenshiftService{routeClient: &routeClient}
	assert.NoError(t, service.UpdateExternalTargetPath(instance, intOrString))
	routes.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

type TestOpenShiftAlternativeSuite struct {
	suite.Suite
}

func (s *TestOpenShiftAlternativeSuite) BeforeTest(_, _ string) {
	err := os.Setenv(deploymentTypeEnvName, deploymentConfigsDeploymentType)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *TestOpenShiftAlternativeSuite) AfterTest(_, _ string) {
	err := os.Unsetenv(deploymentTypeEnvName)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_IsDeploymentReadyErr() {
	t := s.T()
	instance := &nexusApi.Nexus{}
	appClient := &openshiftMock.AppsV1Interface{}
	deploymentConf := &openshiftMock.DeploymentConfigInterface{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.
		On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).
		Return(nil, fmt.Errorf("test"))

	service := OpenshiftService{appClient: appClient}
	_, err := service.IsDeploymentReady(instance)

	assert.Contains(t, err.Error(), "failed to get deployment config")
	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_IsDeploymentReadyFalse() {
	t := s.T()
	deploymentConfInstance := appv1.DeploymentConfig{}
	instance := &nexusApi.Nexus{}
	appClient := &openshiftMock.AppsV1Interface{}
	deploymentConf := &openshiftMock.DeploymentConfigInterface{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)

	service := OpenshiftService{appClient: appClient}

	ok, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.False(t, *ok)
	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_IsDeploymentReadyTrue() {
	t := s.T()
	deploymentConfInstance := appv1.DeploymentConfig{
		Status: appv1.DeploymentConfigStatus{
			UpdatedReplicas:   1,
			AvailableReplicas: 1,
		}}

	instance := &nexusApi.Nexus{}
	appClient := &openshiftMock.AppsV1Interface{}
	deploymentConf := &openshiftMock.DeploymentConfigInterface{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)

	service := OpenshiftService{appClient: appClient}

	ok, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.True(t, *ok)
	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddKeycloakProxyToDeployConf_GetErr() {
	t := s.T()
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	appClient := openshiftMock.AppsV1Interface{}
	deploymentConfig := &openshiftMock.DeploymentConfigInterface{}

	appClient.On("DeploymentConfigs", namespace).Return(deploymentConfig)
	deploymentConfig.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, fmt.Errorf("test"))

	service := OpenshiftService{appClient: &appClient}

	assert.Error(t, service.AddKeycloakProxyToDeployConf(instance, nil))
	appClient.AssertExpectations(t)
	deploymentConfig.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddKeycloakProxyToDeployConf_AlreadyExist() {
	t := s.T()
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	containerSpec := createContainer(instance)
	deploymentConfInstance := createDeploymentConfInstance([]coreV1Api.Container{containerSpec})
	appClient := openshiftMock.AppsV1Interface{}
	deploymentConfig := &openshiftMock.DeploymentConfigInterface{}

	appClient.On("DeploymentConfigs", namespace).Return(deploymentConfig)
	deploymentConfig.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)

	service := OpenshiftService{appClient: &appClient}

	assert.NoError(t, service.AddKeycloakProxyToDeployConf(instance, nil))
	appClient.AssertExpectations(t)
	deploymentConfig.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddKeycloakProxyToDeployConf_UpdateErr() {
	t := s.T()
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	containerSpec := createContainer(instance)
	deploymentConfInstance := createDeploymentConfInstance([]coreV1Api.Container{})
	expectedDeploymentConfInstance := createDeploymentConfInstance([]coreV1Api.Container{containerSpec})
	errTest := fmt.Errorf("test")

	appClient := openshiftMock.AppsV1Interface{}
	deploymentConfig := &openshiftMock.DeploymentConfigInterface{}

	appClient.On("DeploymentConfigs", namespace).Return(deploymentConfig)
	deploymentConfig.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)
	deploymentConfig.
		On("Update", context.TODO(), &expectedDeploymentConfInstance, metav1.UpdateOptions{}).
		Return(nil, errTest)

	service := OpenshiftService{appClient: &appClient}
	err := service.AddKeycloakProxyToDeployConf(instance, nil)

	assert.Contains(t, err.Error(), "failed to update deployment name")
	appClient.AssertExpectations(t)
	deploymentConfig.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddKeycloakProxyToDeployConf() {
	t := s.T()
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	containerSpec := createContainer(instance)
	deploymentConfInstance := createDeploymentConfInstance([]coreV1Api.Container{})
	expectedDeploymentConfInstance := createDeploymentConfInstance([]coreV1Api.Container{containerSpec})
	appClient := openshiftMock.AppsV1Interface{}
	deploymentConfig := &openshiftMock.DeploymentConfigInterface{}

	appClient.On("DeploymentConfigs", namespace).Return(deploymentConfig)
	deploymentConfig.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)
	deploymentConfig.
		On("Update", context.TODO(), &expectedDeploymentConfInstance, metav1.UpdateOptions{}).
		Return(nil, nil)

	service := OpenshiftService{appClient: &appClient}
	assert.NoError(t, service.AddKeycloakProxyToDeployConf(instance, nil))
	appClient.AssertExpectations(t)
	deploymentConfig.AssertExpectations(t)
}

func TestOpenshiftTestSuite(t *testing.T) {
	suite.Run(t, new(TestOpenShiftAlternativeSuite))
}
