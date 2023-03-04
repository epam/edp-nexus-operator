package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	nexusApi "github.com/epam/edp-nexus-operator/v2/api/v1"
	pMock "github.com/epam/edp-nexus-operator/v2/mocks/platform"
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
	ready, err := nexusService.IsDeploymentReady(&instance)
	assert.Contains(t, err.Error(), "failed to check if deployment is ready")
	assert.False(t, *ready)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_LocalGetNexusRestApiUrlErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	nexusService := ServiceImpl{
		platformService:      &platformMock,
		runningInClusterFunc: ReturnFalse,
	}
	errTest := errors.New("test")
	platformMock.On("GetExternalUrl", namespace, name).Return("", "", "", errTest)

	_, err := nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Nexus REST API URL")
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_LocalGetSecretDataErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	assert.Contains(t, err.Error(), "failed to get secret")
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_GetConfigMapDataErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	assert.Contains(t, err.Error(), "failed to get default tasks from Config Map")
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_UnmarshalErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	assert.Contains(t, err.Error(), "secret")
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_CreateJenkinsServiceAccountErr(t *testing.T) {
	scheme := runtime.NewScheme()
	err := jenkinsApi.AddToScheme(scheme)
	assert.NoError(t, err)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&jenkinsApi.Jenkins{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "jenkins",
				},
			},
		).
		Build()

	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	assert.Contains(t, err.Error(), "failed to create Jenkins service account")
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_GetSecretDataErr2(t *testing.T) {
	scheme := runtime.NewScheme()
	err := jenkinsApi.AddToScheme(scheme)
	assert.NoError(t, err)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&jenkinsApi.Jenkins{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "jenkins",
				},
			},
		).
		Build()

	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	assert.Contains(t, err.Error(), "failed to get CI user credentials")
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_RunScriptErr(t *testing.T) {
	scheme := runtime.NewScheme()
	err := jenkinsApi.AddToScheme(scheme)
	assert.NoError(t, err)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&jenkinsApi.Jenkins{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "jenkins",
				},
			},
		).
		Build()

	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	platformMock.
		On("GetConfigMapData", namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).
		Return(configData, nil)
	platformMock.On("CreateSecret", instance, newName).Return(nil)
	platformMock.On("CreateJenkinsServiceAccount", namespace, newName).Return(nil)
	platformMock.On("GetSecretData", namespace, newName).Return(secretData, nil)

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user - name")
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_ExposeConfiguration_createEDPComponentErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance).Build()

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
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta(),
		Spec: nexusApi.NexusSpec{KeycloakSpec: nexusApi.KeycloakSpec{Enabled: true}}}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance).Build()

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

	errTest := errors.New("UnableToGetExternalURL")

	configData := map[string]string{nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix: string(raw)}

	platformMock.On("GetSecretData", namespace, secretName).Return(secretData, nil)
	platformMock.On("GetConfigMapData", namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix)).Return(configData, nil)
	platformMock.On("GetExternalUrl", namespace, name).Return(host, "", "", errTest).Once()

	_, err = nexusService.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get external URL: UnableToGetExternalURL")
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Configure_CreateSecretErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	platformMock := pMock.PlatformService{}
	errTest := errors.New("test")

	platformMock.On("CreateSecret", instance, instance.Name+"-admin-password").Return(errTest)

	nexusService := ServiceImpl{platformService: &platformMock}
	configure, ok, err := nexusService.Configure(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create secret")
	assert.False(t, ok)
	assert.Equal(t, instance, configure)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Configure_getNexusAdminPasswordErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	assert.Contains(t, err.Error(), "failed to get Nexus admin password from secret")
	assert.False(t, ok)
	assert.Equal(t, instance, configure)
	platformMock.AssertExpectations(t)
}

func TestServiceImpl_Configure_IsNexusRestApiReadyErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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
	assert.Contains(t, err.Error(), "checking if Nexus REST API is ready has been failed")
	assert.False(t, ok)
	assert.Equal(t, instance, configure)
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
	assert.Contains(t, err.Error(), "failed to get nexus owner")
}

func TestServiceImpl_ClientForNexusChild_getNexusAdminPasswordErr(t *testing.T) {
	ctx := context.Background()
	nexusUser := nexusApi.NexusUser{
		ObjectMeta: ObjectMeta(),
		Spec:       nexusApi.NexusUserSpec{OwnerName: name},
	}
	nexusCR := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &nexusApi.Nexus{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(nexusCR).Build()

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
	assert.Contains(t, err.Error(), "failed to get nexus admin password")
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
	instance := &nexusApi.Nexus{ObjectMeta: ObjectMeta()}
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

	nexusClient.
		On("RunScript", "disable-outreach-capability", scriptData).
		Return([]byte{}, nil).
		Once()

	platformMock.
		On("GetConfigMapData", instance.Namespace, "name-default-capabilities").
		Return(map[string]string{
			"default-capabilities": "[]",
		}, nil)
	nexusClient.
		On("RunScript", "enable-realm", map[string]interface{}{
			"name": "NuGetApiKey",
		}).
		Return([]byte{}, nil).
		Once()
	platformMock.On("GetConfigMapData", instance.Namespace, "name-default-roles").Return(map[string]string{
		"default-roles": "[]",
	}, nil)

	platformMock.On("GetConfigMapData", instance.Namespace, "name-blobs").Return(map[string]string{
		"blobs": "[]",
	}, nil)
	platformMock.
		On("GetConfigMapData", instance.Namespace, "name-repos-to-create").
		Return(map[string]string{nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix: "[]"}, nil)
	platformMock.
		On("GetConfigMapData", instance.Namespace, "name-repos-to-delete").
		Return(map[string]string{nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix: "[]"}, nil)
	nexusClient.
		On("RunScript", "setup-anonymous-access", map[string]interface{}{"anonymous_access": false}).
		Return([]byte{}, nil).
		Once()

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
