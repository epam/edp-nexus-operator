package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	keycloakHelper "github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	coreV1Api "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pMock "github.com/epam/edp-nexus-operator/v2/mocks/platform"
	nexusApi "github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
)

const (
	name      = "name"
	namespace = "namespace"
	URLScheme = "https"
	host      = "domain"
)

func ObjectMeta() v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func createSecretData() map[string][]byte {
	return map[string][]byte{
		"password": {'p'},
	}
}

func ReturnTrue() bool {
	return true
}

func ReturnFalse() bool {
	return false
}

func TestServiceImpl_IsDeploymentReadyError(t *testing.T) {
	instance := nexusApi.Nexus{}
	platformMock := pMock.PlatformService{}
	ok := false
	errTest := errors.New("test")
	platformMock.On("IsDeploymentReady", instance).Return(&ok, errTest)

	nexusService := ServiceImpl{platformService: &platformMock}
	ready, err := nexusService.IsDeploymentReady(instance)
	assert.Equal(t, errTest, err)
	assert.False(t, *ready)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Integration_GetKeycloakClientErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}}}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{platformService: &platformMock}
	errTest := errors.New("test")
	keycloak := keycloakApi.KeycloakClient{}

	platformMock.On("GetKeycloakClient", name, namespace).Return(keycloak, errTest)

	_, err := nexusService.Integration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get Keycloak client data!"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Integration_CantGetOwnerKeycloakRealm(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}},
	}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{platformService: &platformMock}
	keycloakClient := keycloakApi.KeycloakClient{}

	platformMock.On("GetKeycloakClient", name, namespace).Return(keycloakClient, nil)

	_, err := nexusService.Integration(instance)
	assert.NoError(t, err)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Integration_GetOwnerKeycloakErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}},
	}
	platformMock := pMock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakApi.KeycloakRealm{})

	realm := keycloakApi.KeycloakRealm{ObjectMeta: ObjectMeta()}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&realm).Build()
	helper := keycloakHelper.MakeHelper(client, scheme, logr.Discard())
	nexusService := ServiceImpl{
		platformService: &platformMock,
		keycloakHelper:  helper,
	}

	ownerRef := v1.OwnerReference{
		Kind: "KeycloakRealm",
		Name: name,
	}

	keycloakClient := keycloakApi.KeycloakClient{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{ownerRef},
			Namespace:       namespace,
			Name:            name,
		},
	}

	platformMock.On("GetKeycloakClient", name, namespace).Return(keycloakClient, nil)

	_, err := nexusService.Integration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get owner for"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Integration_GetExternalUrlErr(t *testing.T) {
	ownerRefRealm := v1.OwnerReference{
		Kind: "KeycloakRealm",
		Name: name,
	}
	ownerRef := v1.OwnerReference{
		Kind: "Keycloak",
		Name: name,
	}

	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}}}
	platformMock := pMock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakApi.KeycloakRealm{}, &keycloakApi.Keycloak{})

	realm := keycloakApi.KeycloakRealm{
		ObjectMeta: v1.ObjectMeta{
			Namespace:       namespace,
			Name:            name,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
	}
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&realm, &keycloak).Build()
	helper := keycloakHelper.MakeHelper(client, scheme, logr.Discard())
	nexusService := ServiceImpl{
		platformService: &platformMock,
		keycloakHelper:  helper,
	}

	keycloakClient := keycloakApi.KeycloakClient{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{ownerRefRealm},
			Namespace:       namespace,
			Name:            name,
		},
	}
	errTest := errors.New("test")

	platformMock.On("GetKeycloakClient", name, namespace).Return(keycloakClient, nil)
	platformMock.On("GetExternalUrl", namespace, name).Return("", host, URLScheme, errTest)

	_, err := nexusService.Integration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get route"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Integration_AddKeycloakProxyToDeployConfErr(t *testing.T) {
	ownerRefRealm := v1.OwnerReference{
		Kind: "KeycloakRealm",
		Name: name,
	}
	ownerRef := v1.OwnerReference{
		Kind: "Keycloak",
		Name: name,
	}

	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}}}
	platformMock := pMock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakApi.KeycloakRealm{}, &keycloakApi.Keycloak{})

	realm := keycloakApi.KeycloakRealm{
		ObjectMeta: v1.ObjectMeta{
			Namespace:       namespace,
			Name:            name,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
	}
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&realm, &keycloak).Build()
	helper := keycloakHelper.MakeHelper(client, scheme, logr.Discard())
	nexusService := ServiceImpl{
		platformService: &platformMock,
		keycloakHelper:  helper}

	keycloakClient := keycloakApi.KeycloakClient{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{ownerRefRealm},
			Namespace:       namespace,
			Name:            name,
		},
	}
	errTest := errors.New("test")

	data := []string{"--skip-openid-provider-tls-verify=true", "--discovery-url=/auth/realms/",
		"--client-id=", "--client-secret=42", "--listen=0.0.0.0:3000", "--redirection-url=https://domain",
		"--upstream-url=http://127.0.0.1:8081"}
	platformMock.On("GetKeycloakClient", name, namespace).Return(keycloakClient, nil)
	platformMock.On("GetExternalUrl", namespace, name).Return("", host, URLScheme, nil)
	platformMock.On("AddKeycloakProxyToDeployConf", instance, data).Return(errTest)

	_, err := nexusService.Integration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to add Keycloak proxy"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Integration_AddPortToServiceErr(t *testing.T) {
	ownerRefRealm := v1.OwnerReference{
		Kind: "KeycloakRealm",
		Name: name,
	}
	ownerRef := v1.OwnerReference{
		Kind: "Keycloak",
		Name: name,
	}

	instance := nexusApi.Nexus{
		ObjectMeta: ObjectMeta(),
		Spec:       nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}},
	}
	platformMock := pMock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakApi.KeycloakRealm{}, &keycloakApi.Keycloak{})

	realm := keycloakApi.KeycloakRealm{
		ObjectMeta: v1.ObjectMeta{
			Namespace:       namespace,
			Name:            name,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
	}
	keycloak := keycloakApi.Keycloak{ObjectMeta: ObjectMeta()}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&realm, &keycloak).Build()
	helper := keycloakHelper.MakeHelper(client, scheme, logr.Discard())
	nexusService := ServiceImpl{
		platformService: &platformMock,
		keycloakHelper:  helper}

	keycloakClient := keycloakApi.KeycloakClient{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{ownerRefRealm},
			Namespace:       namespace,
			Name:            name,
		},
	}
	errTest := errors.New("test")

	data := []string{"--skip-openid-provider-tls-verify=true", "--discovery-url=/auth/realms/",
		"--client-id=", "--client-secret=42", "--listen=0.0.0.0:3000", "--redirection-url=https://domain",
		"--upstream-url=http://127.0.0.1:8081"}
	keyCloakProxyPort := coreV1Api.ServicePort{
		Name:       "keycloak-proxy",
		Port:       nexusDefaultSpec.NexusKeycloakProxyPort,
		Protocol:   coreV1Api.ProtocolTCP,
		TargetPort: intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort},
	}

	platformMock.On("GetKeycloakClient", name, namespace).Return(keycloakClient, nil)
	platformMock.On("GetExternalUrl", namespace, name).Return("", host, URLScheme, nil)
	platformMock.On("AddKeycloakProxyToDeployConf", instance, data).Return(nil)
	platformMock.On("AddPortToService", instance, keyCloakProxyPort).Return(errTest)

	_, err := nexusService.Integration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to add Keycloak proxy port to service"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Integration_UpdateExternalTargetPathErr(t *testing.T) {
	ownerRefRealm := v1.OwnerReference{
		Kind: "KeycloakRealm",
		Name: name,
	}
	ownerRef := v1.OwnerReference{
		Kind: "Keycloak",
		Name: name,
	}

	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}}}
	platformMock := pMock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakApi.KeycloakRealm{}, &keycloakApi.Keycloak{})

	realm := keycloakApi.KeycloakRealm{
		ObjectMeta: v1.ObjectMeta{
			Namespace:       namespace,
			Name:            name,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
	}
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&realm, &keycloak).Build()
	helper := keycloakHelper.MakeHelper(client, scheme, logr.Discard())
	nexusService := ServiceImpl{
		platformService: &platformMock,
		keycloakHelper:  helper}

	keycloakClient := keycloakApi.KeycloakClient{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{ownerRefRealm},
			Namespace:       namespace,
			Name:            name,
		},
	}
	errTest := errors.New("test")

	data := []string{"--skip-openid-provider-tls-verify=true", "--discovery-url=/auth/realms/",
		"--client-id=", "--client-secret=42", "--listen=0.0.0.0:3000", "--redirection-url=https://domain",
		"--upstream-url=http://127.0.0.1:8081"}
	keyCloakProxyPort := coreV1Api.ServicePort{
		Name:       "keycloak-proxy",
		Port:       nexusDefaultSpec.NexusKeycloakProxyPort,
		Protocol:   coreV1Api.ProtocolTCP,
		TargetPort: intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort},
	}

	platformMock.On("GetKeycloakClient", name, namespace).Return(keycloakClient, nil)
	platformMock.On("GetExternalUrl", namespace, name).Return("", host, URLScheme, nil)
	platformMock.On("AddKeycloakProxyToDeployConf", instance, data).Return(nil)
	platformMock.On("AddPortToService", instance, keyCloakProxyPort).Return(nil)
	platformMock.On("UpdateExternalTargetPath", instance, intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort}).Return(errTest)

	_, err := nexusService.Integration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to update target port in Route"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Integration(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	nexusService := ServiceImpl{}

	_, err := nexusService.Integration(instance)
	assert.NoError(t, err)
}

func TestServiceImpl_ExposeConfiguration_LocalGetNexusRestApiUrlErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnFalse,
	}
	errTest := errors.New("test")
	platformMock.On("GetExternalUrl", namespace, name).Return("", "", "", errTest)

	_, err := nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get Nexus REST API URL"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_LocalGetSecretDataErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnFalse,
	}
	errTest := errors.New("test")
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	platformMock.On("GetExternalUrl", namespace, name).Return("", "", "", nil)
	platformMock.On("GetSecretData", namespace, secretName).Return(nil, errTest)

	_, err := nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get Secret"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_GetConfigMapDataErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
	}
	errTest := errors.New("test")
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()

	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(nil, errTest)

	_, err := nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get default tasks from Config Map"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_UnmarshalErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
	}
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()
	configData := map[string]string{"nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix": ""}

	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)

	_, err := nexusService.ExposeConfiguration(context.Background(), instance)
	errJSON := &json.SyntaxError{}
	assert.ErrorAs(t, err, &errJSON)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_CreateSecretErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
	}
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()
	parseUsers := []map[string]interface{}{{"username": name, "first_name": name, "last_name": name}}
	raw, err := json.Marshal(parseUsers)
	if err != nil {
		t.Fatal(err)
	}
	errTest := errors.New("test")

	configData := map[string]string{nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix: string(raw)}

	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)
	platformMock.On("CreateSecret", instance, fmt.Sprintf("%s-%s", name, name)).Return(errTest)

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "secret"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_CreateJenkinsServiceAccountErr(t *testing.T) {
	scheme := runtime.NewScheme()
	err := jenkinsV1Api.AddToScheme(scheme)
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&jenkinsV1Api.Jenkins{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "jenkins",
				},
			},
		).
		Build()

	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
		client:               k8sClient,
	}
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()
	parseUsers := []map[string]interface{}{{"username": name, "first_name": name, "last_name": name}}
	raw, err := json.Marshal(parseUsers)
	if err != nil {
		t.Fatal(err)
	}
	errTest := errors.New("test")

	configData := map[string]string{nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix: string(raw)}

	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)
	platformMock.On("CreateSecret", instance, fmt.Sprintf("%s-%s", name, name)).Return(nil)
	platformMock.On("CreateJenkinsServiceAccount", namespace, fmt.Sprintf("%s-%s", name, name)).Return(errTest)

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to create Jenkins service account"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_GetSecretDataErr2(t *testing.T) {
	scheme := runtime.NewScheme()
	err := jenkinsV1Api.AddToScheme(scheme)
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&jenkinsV1Api.Jenkins{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "jenkins",
				},
			},
		).
		Build()

	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
		client:               k8sClient,
	}
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()
	parseUsers := []map[string]interface{}{{"username": name, "first_name": name, "last_name": name}}
	raw, err := json.Marshal(parseUsers)
	if err != nil {
		t.Fatal(err)
	}
	errTest := errors.New("test")

	configData := map[string]string{nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix: string(raw)}

	newName := fmt.Sprintf("%s-%s", name, name)
	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)
	platformMock.On("CreateSecret", instance, newName).Return(nil)
	platformMock.On("CreateJenkinsServiceAccount", namespace, newName).Return(nil)
	platformMock.On("GetSecretData", namespace, newName).Return(nil, errTest)

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get CI user credentials"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_RunScriptErr(t *testing.T) {
	scheme := runtime.NewScheme()
	err := jenkinsV1Api.AddToScheme(scheme)
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&jenkinsV1Api.Jenkins{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "jenkins",
				},
			},
		).
		Build()

	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
		client:               k8sClient,
	}
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()
	parseUsers := []map[string]interface{}{{"username": name, "first_name": name, "last_name": name}}
	raw, err := json.Marshal(parseUsers)
	if err != nil {
		t.Fatal(err)
	}

	configData := map[string]string{nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix: string(raw)}

	newName := fmt.Sprintf("%s-%s", name, name)
	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)
	platformMock.On("CreateSecret", instance, newName).Return(nil)
	platformMock.On("CreateJenkinsServiceAccount", namespace, newName).Return(nil)
	platformMock.On("GetSecretData", namespace, newName).Return(secretData, nil)

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to create user name"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_createEDPComponentErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&instance).Build()

	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		client:               client,
		runningInClusterFunc: ReturnTrue,
	}
	platformMock.On("GetExternalUrl", namespace, name).Return(host, "", "", nil)

	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()
	var parseUsers []map[string]interface{}
	raw, err := json.Marshal(parseUsers)
	if err != nil {
		t.Fatal(err)
	}

	configData := map[string]string{nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix: string(raw)}

	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_GetExternalUrlErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}}}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&instance).Build()

	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		client:               client,
		runningInClusterFunc: ReturnTrue,
	}
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()
	var parseUsers []map[string]interface{}
	raw, err := json.Marshal(parseUsers)
	if err != nil {
		t.Fatal(err)
	}
	errTest := errors.New("test")

	configData := map[string]string{nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix: string(raw)}

	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)
	platformMock.On("GetExternalUrl", namespace, name).Return(host, "", "", errTest).Once()

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get route from cluster"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_CantCreateKeycloakClient(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}}}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&instance).Build()

	platformMock := pMock.PlatformService{}
	platformMock.On("GetExternalUrl", namespace, name).Return(host, "", "", nil)
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		client:               client,
		runningInClusterFunc: ReturnTrue,
	}
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := createSecretData()
	var parseUsers []map[string]interface{}
	raw, err := json.Marshal(parseUsers)
	if err != nil {
		t.Fatal(err)
	}

	keycloakClient := keycloakApi.KeycloakClient{ObjectMeta: ObjectMeta(),
		Spec: keycloakApi.KeycloakClientSpec{
			ClientId:            instance.Name,
			Public:              true,
			WebUrl:              host,
			DefaultClientScopes: []string{"edp"},
		}}

	errTest := errors.New("test")
	configData := map[string]string{nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix: string(raw)}

	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)
	platformMock.On("CreateKeycloakClient", &keycloakClient).Return(errTest)

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.NoError(t, err)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Configure_CreateSecretErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	errTest := errors.New("test")

	platformMock.On("CreateSecret", instance, instance.Name+"-admin-password").Return(errTest)

	nexusService := ServiceImpl{platformService: &platformMock}
	configure, ok, err := nexusService.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to create Secret"))
	assert.False(t, ok)
	assert.Equal(t, &instance, configure)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Configure_getNexusAdminPasswordErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	errTest := errors.New("test")

	platformMock.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	platformMock.On("GetSecretData", namespace, secretName).Return(nil, errTest)

	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
	}
	configure, ok, err := nexusService.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to get Nexus admin password from secret"))
	assert.False(t, ok)
	assert.Equal(t, &instance, configure)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Configure_IsNexusRestApiReadyErr(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	secretData := createSecretData()

	platformMock.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)

	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
		clientBuilder: func(url string, user string, password string) Client {
			return nexus.Init(url, user, password)
		},
	}
	configure, ok, err := nexusService.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "checking if Nexus REST API is ready has been failed"))
	assert.False(t, ok)
	assert.Equal(t, &instance, configure)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ClientForNexusChild_ClientErr(t *testing.T) {
	ctx := context.Background()
	nexusUser := nexusApi.NexusUser{ObjectMeta: ObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	nexusService := ServiceImpl{client: client}
	child, err := nexusService.ClientForNexusChild(ctx, &nexusUser)
	assert.Nil(t, child)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unable to get nexus owner"))
}

func TestServiceImpl_ClientForNexusChild_getNexusAdminPasswordErr(t *testing.T) {
	ctx := context.Background()
	nexusUser := nexusApi.NexusUser{
		ObjectMeta: ObjectMeta(),
		Spec:       nexusApi.NexusUserSpec{OwnerName: name},
	}
	nexusCR := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&nexusCR).Build()

	errTest := errors.New("test")
	secretName := fmt.Sprintf("%v-admin-password", nexusUser.Name)

	platformMock := pMock.PlatformService{}
	platformMock.On("GetSecretData", namespace, secretName).Return(nil, errTest)

	nexusService := ServiceImpl{
		client:          client,
		platformService: &platformMock}
	child, err := nexusService.ClientForNexusChild(ctx, &nexusUser)
	assert.Nil(t, child)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unable to get nexus admin password"))
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ClientForNexusChild(t *testing.T) {
	ctx := context.Background()
	nexusUser := nexusApi.NexusUser{
		ObjectMeta: ObjectMeta(),
		Spec:       nexusApi.NexusUserSpec{OwnerName: name},
	}
	nexusCR := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&nexusCR).Build()

	secretName := fmt.Sprintf("%v-admin-password", nexusUser.Name)
	secretData := createSecretData()

	platformMock := pMock.PlatformService{}
	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)

	nexusService := ServiceImpl{
		client:               client,
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
	}
	child, err := nexusService.ClientForNexusChild(ctx, &nexusUser)
	assert.NotNil(t, child)
	assert.NoError(t, err)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Configure(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	secretData := createSecretData()

	platformMock.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)

	nexusClient := nexus.Mock{}
	nexusClient.On("IsNexusRestApiReady").Return(true, 0, nil)
	platformMock.On("GetConfigMapData", instance.Namespace, "name-scripts").Return(map[string]string{}, nil)
	nexusClient.On("DeclareDefaultScripts", map[string]string{}).Return(nil)
	nexusClient.On("AreDefaultScriptsDeclared", map[string]string{}).Return(true, nil)
	platformMock.On("GetConfigMapData", instance.Namespace, "name-default-tasks").Return(map[string]string{
		nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix: "[]",
	}, nil)

	var scriptData map[string]interface{}
	nexusClient.On("RunScript", "disable-outreach-capability", scriptData).
		Return([]byte{}, nil).Once()

	platformMock.On("GetConfigMapData", instance.Namespace, "name-default-capabilities").
		Return(map[string]string{
			"default-capabilities": "[]",
		}, nil)
	nexusClient.On("RunScript", "enable-realm", map[string]interface{}{
		"name": "NuGetApiKey",
	}).
		Return([]byte{}, nil).Once()
	platformMock.On("GetConfigMapData", instance.Namespace, "name-default-roles").Return(map[string]string{
		"default-roles": "[]",
	}, nil)

	platformMock.On("GetConfigMapData", instance.Namespace, "name-blobs").Return(map[string]string{
		"blobs": "[]",
	}, nil)
	platformMock.On("GetConfigMapData", instance.Namespace, "name-repos-to-create").Return(map[string]string{
		nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix: "[]",
	}, nil)
	platformMock.On("GetConfigMapData", instance.Namespace, "name-repos-to-delete").Return(map[string]string{
		nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix: "[]",
	}, nil)
	nexusClient.On("RunScript", "setup-anonymous-access",
		map[string]interface{}{"anonymous_access": false}).
		Return([]byte{}, nil).Once()

	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnTrue,
		clientBuilder: func(url string, user string, password string) Client {
			return &nexusClient
		},
	}
	_, _, err := nexusService.Configure(instance)
	assert.NoError(t, err)
	platformMock.AssertExpectations(t)
	nexusClient.AssertExpectations(t)
}
