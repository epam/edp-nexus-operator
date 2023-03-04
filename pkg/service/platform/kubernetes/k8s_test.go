package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	edpCompApi "github.com/epam/edp-component-operator/api/v1"
	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	nexusApi "github.com/epam/edp-nexus-operator/v2/api/v1"
	kMock "github.com/epam/edp-nexus-operator/v2/mocks/kubernetes"
	platformHelper "github.com/epam/edp-nexus-operator/v2/pkg/service/platform/helper"
)

const (
	name      = "name"
	namespace = "ns"
	host      = "host"
)

func createObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func TestK8SService_IsDeploymentReadyErr(t *testing.T) {
	ctx := context.TODO()
	errTest := fmt.Errorf("test")
	instance := &nexusApi.Nexus{}
	appV1Client := kMock.AppsV1Interface{}
	deployment := &kMock.DeploymentInterface{}
	service := K8SService{
		appClient: &appV1Client,
	}

	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(nil, errTest)

	ready, err := service.IsDeploymentReady(instance)
	assert.Error(t, err)
	assert.Nil(t, ready)
}

func TestK8SService_IsDeploymentReadyFalse(t *testing.T) {
	ctx := context.TODO()
	instance := &nexusApi.Nexus{}
	deploymentInstance := &appsv1.Deployment{}
	appV1Client := kMock.AppsV1Interface{}
	deployment := &kMock.DeploymentInterface{}
	service := K8SService{
		appClient: &appV1Client,
	}

	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(deploymentInstance, nil)

	ready, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.False(t, *ready)
}

func TestK8SService_IsDeploymentReadyTrue(t *testing.T) {
	ctx := context.TODO()
	instance := &nexusApi.Nexus{}
	deploymentInstance := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			UpdatedReplicas:   1,
			AvailableReplicas: 1,
		}}
	appV1Client := kMock.AppsV1Interface{}
	deployment := &kMock.DeploymentInterface{}
	service := K8SService{
		appClient: &appV1Client,
	}

	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(deploymentInstance, nil)

	ready, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.True(t, *ready)
}

func TestK8SService_GetExternalUrl_GetErr(t *testing.T) {
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, fmt.Errorf("test"))

	// nolint
	_, _, _, err := service.GetExternalUrl(namespace, name)
	assert.Contains(t, err.Error(), "failed to get ingress")
}

func TestK8SService_GetExternalUrl(t *testing.T) {
	value := networkingV1.IngressRuleValue{HTTP: &networkingV1.HTTPIngressRuleValue{
		Paths: []networkingV1.HTTPIngressPath{{
			Path: name,
		}},
	}}
	ingressInstance := networkingV1.Ingress{
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{{
				IngressRuleValue: value,
				Host:             host,
			},
			},
		},
	}
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&ingressInstance, nil)

	url, s, s2, err := service.GetExternalUrl(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, "https://hostname", url)
	assert.Equal(t, host, s)
	assert.Equal(t, "https", s2)
}

func TestK8SService_CreateEDPComponentIfNotExist_GetErr(t *testing.T) {
	instance := nexusApi.Nexus{}
	client := fake.NewClientBuilder().Build()

	service := K8SService{client: client}

	err := service.CreateEDPComponentIfNotExist(&instance, "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no kind is registered")
}

func TestK8SService_CreateEDPComponentIfNotExist_AlreadyExist(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	EDPComponent := edpCompApi.EDPComponent{ObjectMeta: createObjectMeta()}
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &edpCompApi.EDPComponent{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&EDPComponent).Build()
	service := K8SService{client: client}

	assert.NoError(t, service.CreateEDPComponentIfNotExist(&instance, "", ""))
}

func TestK8SService_CreateEDPComponentIfNotExist(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &edpCompApi.EDPComponent{}, &nexusApi.Nexus{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	service := K8SService{client: client, Scheme: scheme}

	assert.NoError(t, service.CreateEDPComponentIfNotExist(&instance, "test.com", "icon.png"))
}

func TestK8SService_CreateJenkinsServiceAccount_BadClient(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion)

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	service := K8SService{client: client, Scheme: scheme}

	assert.Error(t, service.CreateJenkinsServiceAccount(namespace, name))
}

func TestK8SService_CreateJenkinsServiceAccount(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &jenkinsV1Api.JenkinsServiceAccount{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	service := K8SService{client: client, Scheme: scheme}
	tmp := &jenkinsV1Api.JenkinsServiceAccount{}

	assert.NoError(t, service.CreateJenkinsServiceAccount(namespace, name))
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, tmp))
}

func TestK8SService_CreateJenkinsServiceAccount_AlreadyExist(t *testing.T) {
	tmp := &jenkinsV1Api.JenkinsServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &jenkinsV1Api.JenkinsServiceAccount{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tmp).Build()
	service := K8SService{client: client, Scheme: scheme}

	assert.NoError(t, service.CreateJenkinsServiceAccount(namespace, name))
}

func TestK8SService_GetSecret(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	secret := &coreV1Api.Secret{}
	service := K8SService{
		CoreClient: &coreClient,
	}

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(secret, nil)

	result, err := service.GetSecret(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, secret, result)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_UpdateSecret(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	secret := &coreV1Api.Secret{ObjectMeta: createObjectMeta()}
	service := K8SService{
		CoreClient: &coreClient,
	}

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Update", context.TODO(), secret, metav1.UpdateOptions{}).Return(nil, nil)

	assert.NoError(t, service.UpdateSecret(secret))
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_GetSecretData_NotFound(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	service := K8SService{
		CoreClient: &coreClient,
	}

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))

	result, err := service.GetSecretData(namespace, name)
	assert.NoError(t, err)
	assert.Nil(t, result)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_GetSecretData(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	data := map[string][]byte{"str": []byte("str")}
	secret := &coreV1Api.Secret{
		Data: data,
	}
	service := K8SService{
		CoreClient: &coreClient,
	}

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(secret, nil)

	result, err := service.GetSecretData(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, data, result)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_GetConfigMapData_IsNotFound(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	configMaps := &kMock.ConfigMapInterface{}
	service := K8SService{
		CoreClient: &coreClient,
	}

	coreClient.On("ConfigMaps", namespace).Return(configMaps)
	configMaps.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))

	data, err := service.GetConfigMapData(namespace, name)
	assert.NoError(t, err)
	assert.Nil(t, data)
	coreClient.AssertExpectations(t)
	configMaps.AssertExpectations(t)
}

func TestK8SService_GetConfigMapData(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	configMaps := &kMock.ConfigMapInterface{}
	data := map[string]string{"str": "str"}
	configMap := coreV1Api.ConfigMap{Data: data}

	coreClient.On("ConfigMaps", namespace).Return(configMaps)
	configMaps.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&configMap, nil)

	service := K8SService{CoreClient: &coreClient}
	result, err := service.GetConfigMapData(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, data, result)
	coreClient.AssertExpectations(t)
	configMaps.AssertExpectations(t)
}

func TestK8SService_CreateSecret_SetControllerReferenceErr(t *testing.T) {
	scheme := runtime.NewScheme()
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	data := map[string][]byte{"str": []byte("str")}
	service := K8SService{Scheme: scheme}

	err := service.CreateSecret(instance, name, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no kind is registered")
}

func TestK8SService_CreateSecret_AlreadyExist(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &nexusApi.Nexus{})

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, nil)

	data := map[string][]byte{"str": []byte("str")}
	service := K8SService{
		Scheme:     scheme,
		CoreClient: &coreClient,
	}

	assert.NoError(t, service.CreateSecret(instance, name, data))
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_CreateSecret_GetSecretErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	errTest := fmt.Errorf("test")
	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &nexusApi.Nexus{})

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	data := map[string][]byte{"str": []byte("str")}
	service := K8SService{
		Scheme:     scheme,
		CoreClient: &coreClient,
	}

	assert.ErrorIs(t, service.CreateSecret(instance, name, data), errTest)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_CreateSecret_CreateSecretErr(t *testing.T) {
	data := map[string][]byte{"str": []byte("str")}
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	labels := platformHelper.GenerateLabels(instance.Name)
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &nexusApi.Nexus{})

	secretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}
	if err := controllerutil.SetControllerReference(instance, secretObject, scheme); err != nil {
		t.Fatal(err)
	}

	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	errTest := fmt.Errorf("test")

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))
	secrets.On("Create", context.TODO(), secretObject, metav1.CreateOptions{}).Return(nil, errTest)

	service := K8SService{
		Scheme:     scheme,
		CoreClient: &coreClient,
	}

	assert.ErrorIs(t, service.CreateSecret(instance, name, data), errTest)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_CreateSecret(t *testing.T) {
	data := map[string][]byte{"str": []byte("str")}
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	labels := platformHelper.GenerateLabels(instance.Name)
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &nexusApi.Nexus{})

	secretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}

	if err := controllerutil.SetControllerReference(instance, secretObject, scheme); err != nil {
		t.Fatal(err)
	}

	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.
		On("Get", context.TODO(), name, metav1.GetOptions{}).
		Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))
	secrets.On("Create", context.TODO(), secretObject, metav1.CreateOptions{}).Return(secretObject, nil)

	service := K8SService{
		Scheme:     scheme,
		CoreClient: &coreClient,
	}

	assert.NoError(t, service.CreateSecret(instance, name, data))
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_Init(t *testing.T) {
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
	service := K8SService{}

	assert.NoError(t, service.Init(restConfig, scheme, client))
}
