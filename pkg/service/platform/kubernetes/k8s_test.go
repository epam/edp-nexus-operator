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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/api/v1/v1"
	nexusApi "github.com/epam/edp-nexus-operator/v2/api/edp/v1"
	kMock "github.com/epam/edp-nexus-operator/v2/mocks/kubernetes"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
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

func TestK8SService_AddKeycloakProxyToDeployConf_GetErr(t *testing.T) {
	ctx := context.TODO()
	instance := &nexusApi.Nexus{}
	deploymentInstance := &appsv1.Deployment{}
	appV1Client := kMock.AppsV1Interface{}
	deployment := &kMock.DeploymentInterface{}
	service := K8SService{
		appClient: &appV1Client,
	}

	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(deploymentInstance, fmt.Errorf("test"))

	err := service.AddKeycloakProxyToDeployConf(instance, nil)
	assert.Contains(t, err.Error(), "failed to get deployment")
}

func TestK8SService_AddKeycloakProxyToDeployConf_UpdateErr(t *testing.T) {
	ctx := context.TODO()
	instance := &nexusApi.Nexus{}
	c := coreV1Api.Container{
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

	deploymentInstance := appsv1.Deployment{}
	deploymentInstanceExpect := appsv1.Deployment{}
	deploymentInstanceExpect.Spec.Template.Spec.Containers = append(deploymentInstanceExpect.Spec.Template.Spec.Containers, c)
	appV1Client := kMock.AppsV1Interface{}
	deployment := &kMock.DeploymentInterface{}
	service := K8SService{
		appClient: &appV1Client,
	}

	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(&deploymentInstance, nil)
	deployment.
		On("Update", ctx, &deploymentInstanceExpect, metav1.UpdateOptions{}).
		Return(nil, fmt.Errorf("test"))

	err := service.AddKeycloakProxyToDeployConf(instance, nil)
	assert.Contains(t, err.Error(), "failed to get deployment")
}

func TestK8SService_AddKeycloakProxyToDeployConf(t *testing.T) {
	ctx := context.TODO()
	instance := &nexusApi.Nexus{}
	c := coreV1Api.Container{
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

	deploymentInstance := appsv1.Deployment{}
	deploymentInstanceExpect := appsv1.Deployment{}
	deploymentInstanceExpect.Spec.Template.Spec.Containers = append(deploymentInstanceExpect.Spec.Template.Spec.Containers, c)
	appV1Client := kMock.AppsV1Interface{}
	deployment := &kMock.DeploymentInterface{}
	service := K8SService{
		appClient: &appV1Client,
	}

	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(&deploymentInstance, nil)
	deployment.On("Update", ctx, &deploymentInstanceExpect, metav1.UpdateOptions{}).Return(nil, nil)

	assert.NoError(t, service.AddKeycloakProxyToDeployConf(instance, nil))
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

func TestK8SService_GetIngressByCr_ListErr(t *testing.T) {
	instance := &nexusApi.Nexus{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
	}
	errTest := fmt.Errorf("test")
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(nil, errTest)

	cr, err := service.GetIngressByCr(instance)
	assert.Error(t, err)
	assert.Nil(t, cr)
	assert.Contains(t, err.Error(), "failed to retrieve ingresses list from the cluster")
}

func TestK8SService_GetIngressByCr_InList(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	ingressInstance := networkingV1.Ingress{ObjectMeta: createObjectMeta()}
	list := networkingV1.IngressList{
		Items: []networkingV1.Ingress{ingressInstance},
	}
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	cr, err := service.GetIngressByCr(instance)
	assert.NoError(t, err)
	assert.Equal(t, &ingressInstance, cr)
}

func TestK8SService_GetIngressByCr_NotInList(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	list := networkingV1.IngressList{
		Items: []networkingV1.Ingress{},
	}
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	cr, err := service.GetIngressByCr(instance)
	assert.NoError(t, err)
	assert.Nil(t, cr)
}

func TestK8SService_CreateKeycloakClientErr(t *testing.T) {
	instance := &keycloakV1Api.KeycloakClient{}
	client := fake.NewClientBuilder().Build()
	service := K8SService{client: client}

	err := service.CreateKeycloakClient(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no kind is registered for the type")
}

func TestK8SService_CreateKeycloakClient(t *testing.T) {
	instance := &keycloakV1Api.KeycloakClient{
		TypeMeta:   metav1.TypeMeta{Kind: "KeycloakClient", APIVersion: "v1"},
		ObjectMeta: createObjectMeta()}
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &keycloakV1Api.KeycloakClient{})

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	service := K8SService{client: client}

	assert.NoError(t, service.CreateKeycloakClient(instance))
}

func TestK8SService_GetKeycloakClientErr(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	service := K8SService{client: client}

	_, err := service.GetKeycloakClient(name, namespace)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no kind is registered for the type")
}

func TestK8SService_GetKeycloakClient(t *testing.T) {
	instance := &keycloakV1Api.KeycloakClient{
		TypeMeta:   metav1.TypeMeta{Kind: "KeycloakClient", APIVersion: "apps/v1"},
		ObjectMeta: createObjectMeta()}
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &keycloakV1Api.KeycloakClient{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance).Build()
	service := K8SService{client: client}

	out, err := service.GetKeycloakClient(name, namespace)
	assert.NoError(t, err)
	assert.Equal(t, *instance, out)
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

func TestK8SService_UpdateExternalTargetPath_GetIngressByCr(t *testing.T) {
	instance := &nexusApi.Nexus{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
	}
	errTest := fmt.Errorf("test")
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}
	orString := intstr.IntOrString{}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(nil, errTest)

	err := service.UpdateExternalTargetPath(instance, orString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get route")
	extClient.AssertExpectations(t)
	ingress.AssertExpectations(t)
}

func TestK8SService_UpdateExternalTargetPath_AlreadyUpdated(t *testing.T) {
	orString := intstr.IntOrString{}
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	path := networkingV1.HTTPIngressPath{
		Backend: networkingV1.IngressBackend{
			Service: &networkingV1.IngressServiceBackend{
				Port: networkingV1.ServiceBackendPort{
					Number: int32(orString.IntValue()),
				},
			},
		},
	}
	rule := networkingV1.IngressRule{
		IngressRuleValue: networkingV1.IngressRuleValue{
			HTTP: &networkingV1.HTTPIngressRuleValue{
				Paths: []networkingV1.HTTPIngressPath{path},
			},
		},
	}
	ingressInstance := networkingV1.Ingress{
		ObjectMeta: createObjectMeta(),
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{rule},
		},
	}
	list := networkingV1.IngressList{
		Items: []networkingV1.Ingress{ingressInstance},
	}
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	assert.NoError(t, service.UpdateExternalTargetPath(instance, orString))
	extClient.AssertExpectations(t)
	ingress.AssertExpectations(t)
}

func TestK8SService_UpdateExternalTargetPath_UpdateErr(t *testing.T) {
	intOrString := intstr.IntOrString{}
	intTwo := intstr.IntOrString{
		Type:   0,
		IntVal: 2,
		StrVal: "",
	}
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	path := networkingV1.HTTPIngressPath{
		Backend: networkingV1.IngressBackend{
			Service: &networkingV1.IngressServiceBackend{
				Port: networkingV1.ServiceBackendPort{
					Number: int32(intTwo.IntValue()),
				},
			},
		},
	}
	rule := networkingV1.IngressRule{
		IngressRuleValue: networkingV1.IngressRuleValue{
			HTTP: &networkingV1.HTTPIngressRuleValue{
				Paths: []networkingV1.HTTPIngressPath{path},
			},
		},
	}
	ingressInstance := networkingV1.Ingress{
		ObjectMeta: createObjectMeta(),
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{rule},
		},
	}
	list := networkingV1.IngressList{
		Items: []networkingV1.Ingress{ingressInstance},
	}
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)
	ingress.On("Update", context.TODO(), &ingressInstance, metav1.UpdateOptions{}).Return(nil, fmt.Errorf("test"))

	err := service.UpdateExternalTargetPath(instance, intOrString)
	assert.Contains(t, err.Error(), "failed to update ingress")
	extClient.AssertExpectations(t)
	ingress.AssertExpectations(t)
}

func TestK8SService_UpdateExternalTargetPath(t *testing.T) {
	intOrString := intstr.IntOrString{}
	intTwo := intstr.IntOrString{
		Type:   0,
		IntVal: 2,
		StrVal: "",
	}
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	path := networkingV1.HTTPIngressPath{
		Backend: networkingV1.IngressBackend{
			Service: &networkingV1.IngressServiceBackend{
				Port: networkingV1.ServiceBackendPort{
					Number: int32(intTwo.IntValue()),
				},
			},
		},
	}
	rule := networkingV1.IngressRule{
		IngressRuleValue: networkingV1.IngressRuleValue{
			HTTP: &networkingV1.HTTPIngressRuleValue{
				Paths: []networkingV1.HTTPIngressPath{path},
			},
		},
	}
	ingressInstance := networkingV1.Ingress{
		ObjectMeta: createObjectMeta(),
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{rule},
		},
	}
	list := networkingV1.IngressList{
		Items: []networkingV1.Ingress{ingressInstance},
	}
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingV1Interface{}
	service := K8SService{
		networkingV1Client: extClient,
	}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)
	ingress.On("Update", context.TODO(), &ingressInstance, metav1.UpdateOptions{}).Return(nil, nil)

	assert.NoError(t, service.UpdateExternalTargetPath(instance, intOrString))
	extClient.AssertExpectations(t)
	ingress.AssertExpectations(t)
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

func TestK8SService_GetServiceByCr_NotFound(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))

	service := K8SService{CoreClient: &coreClient}

	cr, err := service.GetServiceByCr(name, namespace)
	assert.Nil(t, cr)
	assert.Contains(t, err.Error(), "not found")
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_GetServiceByCr_Err(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	errTest := fmt.Errorf("test")
	service := K8SService{CoreClient: &coreClient}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	cr, err := service.GetServiceByCr(name, namespace)
	assert.Nil(t, cr)
	assert.Contains(t, err.Error(), "failed to get service")
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_GetServiceByCr(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	serviceInstance := coreV1Api.Service{}
	service := K8SService{CoreClient: &coreClient}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&serviceInstance, nil)

	cr, err := service.GetServiceByCr(name, namespace)
	assert.Equal(t, &serviceInstance, cr)
	assert.NoError(t, err)
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_AddPortToService_GetServiceByCrErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	portSpec := &coreV1Api.ServicePort{}
	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	errTest := fmt.Errorf("test")
	service := K8SService{CoreClient: &coreClient}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	err := service.AddPortToService(instance, portSpec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get service")
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_AddPortToService_PortInService(t *testing.T) {
	instance := nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	portSpec := coreV1Api.ServicePort{Name: name}

	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	serviceInstance := coreV1Api.Service{
		Spec: coreV1Api.ServiceSpec{
			Ports: []coreV1Api.ServicePort{portSpec},
		},
	}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.Background(), name, metav1.GetOptions{}).Return(&serviceInstance, nil)

	service := K8SService{CoreClient: &coreClient}
	err := service.AddPortToService(&instance, &portSpec)

	assert.NoError(t, err)
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_AddPortToService_UpdateErr(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	portSpec := &coreV1Api.ServicePort{Name: name}
	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	serviceInstance := &coreV1Api.Service{}
	errTest := fmt.Errorf("test")
	service := K8SService{CoreClient: &coreClient}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(serviceInstance, nil)
	services.On("Update", context.TODO(), serviceInstance, metav1.UpdateOptions{}).Return(nil, errTest)

	assert.ErrorIs(t, service.AddPortToService(instance, portSpec), errTest)
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_AddPortToService(t *testing.T) {
	instance := &nexusApi.Nexus{ObjectMeta: createObjectMeta()}
	portSpec := &coreV1Api.ServicePort{Name: name}

	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	serviceInstance := &coreV1Api.Service{}
	service := K8SService{CoreClient: &coreClient}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(serviceInstance, nil)
	services.On("Update", context.TODO(), serviceInstance, metav1.UpdateOptions{}).Return(nil, nil)

	assert.NoError(t, service.AddPortToService(instance, portSpec))
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
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
