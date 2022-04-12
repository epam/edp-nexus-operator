package kubernetes

import (
	"context"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"

	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kMock "github.com/epam/edp-nexus-operator/v2/mocks/kubernetes"
	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1alpha1"
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
	errTest := errors.New("test")
	instance := v1alpha1.Nexus{}
	appV1Client := kMock.AppsV1Client{}
	deployment := &kMock.Deployment{}
	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(nil, errTest)

	service := K8SService{
		appClient: &appV1Client,
	}
	ready, err := service.IsDeploymentReady(instance)
	assert.Error(t, err)
	assert.Nil(t, ready)
}

func TestK8SService_IsDeploymentReadyFalse(t *testing.T) {
	ctx := context.TODO()
	instance := v1alpha1.Nexus{}
	deploymentInstance := &appsv1.Deployment{}
	appV1Client := kMock.AppsV1Client{}
	deployment := &kMock.Deployment{}
	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(deploymentInstance, nil)

	service := K8SService{
		appClient: &appV1Client,
	}
	ready, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.False(t, *ready)
}

func TestK8SService_IsDeploymentReadyTrue(t *testing.T) {
	ctx := context.TODO()
	instance := v1alpha1.Nexus{}
	deploymentInstance := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			UpdatedReplicas:   1,
			AvailableReplicas: 1,
		}}
	appV1Client := kMock.AppsV1Client{}
	deployment := &kMock.Deployment{}
	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(deploymentInstance, nil)

	service := K8SService{
		appClient: &appV1Client,
	}
	ready, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.True(t, *ready)
}

func TestK8SService_AddKeycloakProxyToDeployConf_GetErr(t *testing.T) {
	ctx := context.TODO()
	instance := v1alpha1.Nexus{}
	deploymentInstance := &appsv1.Deployment{}
	appV1Client := kMock.AppsV1Client{}
	deployment := &kMock.Deployment{}
	errTest := errors.New("test")
	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(deploymentInstance, errTest)

	service := K8SService{
		appClient: &appV1Client,
	}
	err := service.AddKeycloakProxyToDeployConf(instance, nil)
	assert.Equal(t, errTest, err)
}

func TestK8SService_AddKeycloakProxyToDeployConf_UpdateErr(t *testing.T) {
	ctx := context.TODO()
	instance := v1alpha1.Nexus{}
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
	errTest := errors.New("test")

	deploymentInstance := appsv1.Deployment{}
	deploymentInstanceExpect := appsv1.Deployment{}
	deploymentInstanceExpect.Spec.Template.Spec.Containers = append(deploymentInstanceExpect.Spec.Template.Spec.Containers, c)
	appV1Client := kMock.AppsV1Client{}
	deployment := &kMock.Deployment{}

	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(&deploymentInstance, nil)
	deployment.On("Update", ctx, &deploymentInstanceExpect, metav1.UpdateOptions{}).Return(nil, errTest)

	service := K8SService{
		appClient: &appV1Client,
	}
	err := service.AddKeycloakProxyToDeployConf(instance, nil)
	assert.Equal(t, errTest, err)
}

func TestK8SService_AddKeycloakProxyToDeployConf(t *testing.T) {
	ctx := context.TODO()
	instance := v1alpha1.Nexus{}
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
	appV1Client := kMock.AppsV1Client{}
	deployment := &kMock.Deployment{}

	appV1Client.On("Deployments", instance.Namespace).Return(deployment)
	deployment.On("Get", ctx, instance.Name, metav1.GetOptions{}).Return(&deploymentInstance, nil)
	deployment.On("Update", ctx, &deploymentInstanceExpect, metav1.UpdateOptions{}).Return(nil, nil)

	service := K8SService{
		appClient: &appV1Client,
	}
	err := service.AddKeycloakProxyToDeployConf(instance, nil)
	assert.NoError(t, err)
}

func TestK8SService_GetExternalUrl_GetErr(t *testing.T) {
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingClient{}
	errTest := errors.New("test")

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	service := K8SService{
		networkingV1Client: extClient,
	}
	_, _, _, err := service.GetExternalUrl(namespace, name)
	assert.Equal(t, errTest, err)
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
	extClient := &kMock.NetworkingClient{}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&ingressInstance, nil)

	service := K8SService{
		networkingV1Client: extClient,
	}
	url, s, s2, err := service.GetExternalUrl(namespace, name)

	assert.NoError(t, err)
	assert.Equal(t, "https://hostname", url)
	assert.Equal(t, host, s)
	assert.Equal(t, "https", s2)
}

func TestK8SService_GetIngressByCr_ListErr(t *testing.T) {
	instance := v1alpha1.Nexus{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
	}
	errTest := errors.New("test")
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingClient{}
	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(nil, errTest)

	service := K8SService{
		networkingV1Client: extClient,
	}
	cr, err := service.GetIngressByCr(instance)
	assert.Error(t, err)
	assert.Nil(t, cr)
	assert.True(t, strings.Contains(err.Error(), "couldn't retrieve ingresses list from the cluster"))
}

func TestK8SService_GetIngressByCr_InList(t *testing.T) {
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	ingressInstance := networkingV1.Ingress{ObjectMeta: createObjectMeta()}
	list := networkingV1.IngressList{
		Items: []networkingV1.Ingress{ingressInstance},
	}
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingClient{}
	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	service := K8SService{
		networkingV1Client: extClient,
	}
	cr, err := service.GetIngressByCr(instance)
	assert.NoError(t, err)
	assert.Equal(t, &ingressInstance, cr)
}

func TestK8SService_GetIngressByCr_NotInList(t *testing.T) {
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	list := networkingV1.IngressList{
		Items: []networkingV1.Ingress{},
	}
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingClient{}
	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	service := K8SService{
		networkingV1Client: extClient,
	}
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
	assert.True(t, strings.Contains(err.Error(), "no kind is registered for the type"))
}

func TestK8SService_CreateKeycloakClient(t *testing.T) {
	instance := &keycloakV1Api.KeycloakClient{
		TypeMeta:   metav1.TypeMeta{Kind: "KeycloakClient", APIVersion: "v1"},
		ObjectMeta: createObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakV1Api.KeycloakClient{})
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	service := K8SService{client: client}
	err := service.CreateKeycloakClient(instance)
	assert.NoError(t, err)
}

func TestK8SService_GetKeycloakClientErr(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	service := K8SService{client: client}
	_, err := service.GetKeycloakClient(name, namespace)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no kind is registered for the type"))
}

func TestK8SService_GetKeycloakClient(t *testing.T) {
	instance := &keycloakV1Api.KeycloakClient{
		TypeMeta:   metav1.TypeMeta{Kind: "KeycloakClient", APIVersion: "apps/v1"},
		ObjectMeta: createObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakV1Api.KeycloakClient{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance).Build()
	service := K8SService{client: client}
	out, err := service.GetKeycloakClient(name, namespace)
	assert.NoError(t, err)
	assert.Equal(t, *instance, out)
}

func TestK8SService_CreateEDPComponentIfNotExist_GetErr(t *testing.T) {
	instance := v1alpha1.Nexus{}
	client := fake.NewClientBuilder().Build()

	service := K8SService{client: client}
	err := service.CreateEDPComponentIfNotExist(instance, "", "")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no kind is registered"))
}

func TestK8SService_CreateEDPComponentIfNotExist_AlreadyExist(t *testing.T) {
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	EDPComponent := edpCompApi.EDPComponent{ObjectMeta: createObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &edpCompApi.EDPComponent{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&EDPComponent).Build()

	service := K8SService{client: client}
	err := service.CreateEDPComponentIfNotExist(instance, "", "")
	assert.NoError(t, err)
}

func TestK8SService_CreateEDPComponentIfNotExist(t *testing.T) {
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &edpCompApi.EDPComponent{}, &v1alpha1.Nexus{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()

	service := K8SService{client: client, Scheme: scheme}
	err := service.CreateEDPComponentIfNotExist(instance, "test.com", "icon.png")
	assert.NoError(t, err)
}

func TestK8SService_CreateJenkinsServiceAccount_BadClient(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion)

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	service := K8SService{client: client, Scheme: scheme}
	err := service.CreateJenkinsServiceAccount(namespace, name)
	assert.Error(t, err)
}

func TestK8SService_CreateJenkinsServiceAccount(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsV1Api.JenkinsServiceAccount{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	service := K8SService{client: client, Scheme: scheme}
	err := service.CreateJenkinsServiceAccount(namespace, name)
	assert.NoError(t, err)
	tmp := &jenkinsV1Api.JenkinsServiceAccount{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, tmp)
	assert.NoError(t, err)
}

func TestK8SService_CreateJenkinsServiceAccount_AlreadyExist(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsV1Api.JenkinsServiceAccount{})
	tmp := &jenkinsV1Api.JenkinsServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tmp).Build()
	service := K8SService{client: client, Scheme: scheme}
	err := service.CreateJenkinsServiceAccount(namespace, name)
	assert.NoError(t, err)
}

func TestK8SService_UpdateExternalTargetPath_GetIngressByCr(t *testing.T) {
	instance := v1alpha1.Nexus{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
	}
	errTest := errors.New("test")
	ingress := &kMock.Ingress{}
	extClient := &kMock.NetworkingClient{}
	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(nil, errTest)

	service := K8SService{
		networkingV1Client: extClient,
	}
	orString := intstr.IntOrString{}
	err := service.UpdateExternalTargetPath(instance, orString)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't get route"))
	extClient.AssertExpectations(t)
	ingress.AssertExpectations(t)
}

func TestK8SService_UpdateExternalTargetPath_AlreadyUpdated(t *testing.T) {
	orString := intstr.IntOrString{}
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
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
	extClient := &kMock.NetworkingClient{}
	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)

	service := K8SService{
		networkingV1Client: extClient,
	}
	err := service.UpdateExternalTargetPath(instance, orString)
	assert.NoError(t, err)
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
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
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
	extClient := &kMock.NetworkingClient{}

	errTest := errors.New("test")
	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)
	ingress.On("Update", context.TODO(), &ingressInstance, metav1.UpdateOptions{}).Return(nil, errTest)

	service := K8SService{
		networkingV1Client: extClient,
	}
	err := service.UpdateExternalTargetPath(instance, intOrString)
	assert.Equal(t, errTest, err)
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
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
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
	extClient := &kMock.NetworkingClient{}

	extClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("List", context.TODO(), metav1.ListOptions{}).Return(&list, nil)
	ingress.On("Update", context.TODO(), &ingressInstance, metav1.UpdateOptions{}).Return(nil, nil)

	service := K8SService{
		networkingV1Client: extClient,
	}
	err := service.UpdateExternalTargetPath(instance, intOrString)
	assert.NoError(t, err)
	extClient.AssertExpectations(t)
	ingress.AssertExpectations(t)
}

func TestK8SService_GetSecret(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	secret := &coreV1Api.Secret{}
	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(secret, nil)
	service := K8SService{
		CoreClient: &coreClient,
	}
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
	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Update", context.TODO(), secret, metav1.UpdateOptions{}).Return(nil, nil)
	service := K8SService{
		CoreClient: &coreClient,
	}
	err := service.UpdateSecret(secret)
	assert.NoError(t, err)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_GetSecretData_NotFound(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}
	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))
	service := K8SService{
		CoreClient: &coreClient,
	}
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
	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(secret, nil)
	service := K8SService{
		CoreClient: &coreClient,
	}
	result, err := service.GetSecretData(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, data, result)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_GetConfigMapData_IsNotFound(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	configMaps := &kMock.ConfigMapInterface{}

	coreClient.On("ConfigMaps", namespace).Return(configMaps)
	configMaps.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))

	service := K8SService{
		CoreClient: &coreClient,
	}
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
	assert.True(t, k8serrors.IsNotFound(err))
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_GetServiceByCr_Err(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	errTest := errors.New("test")

	coreClient.On("Services", namespace).Return(services)

	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	service := K8SService{CoreClient: &coreClient}
	cr, err := service.GetServiceByCr(name, namespace)
	assert.Nil(t, cr)
	assert.Equal(t, errTest, err)
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_GetServiceByCr(t *testing.T) {
	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	serviceInstance := coreV1Api.Service{}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&serviceInstance, nil)

	service := K8SService{CoreClient: &coreClient}
	cr, err := service.GetServiceByCr(name, namespace)
	assert.Equal(t, &serviceInstance, cr)
	assert.NoError(t, err)
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_AddPortToService_GetServiceByCrErr(t *testing.T) {
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	portSpec := coreV1Api.ServicePort{}
	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	errTest := errors.New("test")

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	service := K8SService{CoreClient: &coreClient}
	err := service.AddPortToService(instance, portSpec)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't get"))
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_AddPortToService_PortInService(t *testing.T) {
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	portSpec := coreV1Api.ServicePort{Name: name}

	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	serviceInstance := coreV1Api.Service{
		Spec: coreV1Api.ServiceSpec{
			Ports: []coreV1Api.ServicePort{portSpec},
		},
	}
	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&serviceInstance, nil)

	service := K8SService{CoreClient: &coreClient}
	err := service.AddPortToService(instance, portSpec)
	assert.NoError(t, err)
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_AddPortToService_UpdateErr(t *testing.T) {
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	portSpec := coreV1Api.ServicePort{Name: name}

	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	serviceInstance := &coreV1Api.Service{}
	errTest := errors.New("test")

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(serviceInstance, nil)
	services.On("Update", context.TODO(), serviceInstance, metav1.UpdateOptions{}).Return(nil, errTest)

	service := K8SService{CoreClient: &coreClient}
	err := service.AddPortToService(instance, portSpec)
	assert.Equal(t, errTest, err)
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_AddPortToService(t *testing.T) {
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	portSpec := coreV1Api.ServicePort{Name: name}

	coreClient := kMock.CoreV1Interface{}
	services := &kMock.ServiceInterface{}
	serviceInstance := &coreV1Api.Service{}

	coreClient.On("Services", namespace).Return(services)
	services.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(serviceInstance, nil)
	services.On("Update", context.TODO(), serviceInstance, metav1.UpdateOptions{}).Return(nil, nil)

	service := K8SService{CoreClient: &coreClient}
	err := service.AddPortToService(instance, portSpec)
	assert.NoError(t, err)
	coreClient.AssertExpectations(t)
	services.AssertExpectations(t)
}

func TestK8SService_CreateSecret_SetControllerReferenceErr(t *testing.T) {
	scheme := runtime.NewScheme()
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	data := map[string][]byte{"str": []byte("str")}
	service := K8SService{Scheme: scheme}
	err := service.CreateSecret(instance, name, data)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no kind is registered"))
}

func TestK8SService_CreateSecret_AlreadyExist(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.Nexus{})
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}

	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, nil)

	data := map[string][]byte{"str": []byte("str")}
	service := K8SService{
		Scheme:     scheme,
		CoreClient: &coreClient,
	}
	err := service.CreateSecret(instance, name, data)
	assert.NoError(t, err)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_CreateSecret_GetSecretErr(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.Nexus{})
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	errTest := errors.New("test")

	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	data := map[string][]byte{"str": []byte("str")}
	service := K8SService{
		Scheme:     scheme,
		CoreClient: &coreClient,
	}
	err := service.CreateSecret(instance, name, data)
	assert.Equal(t, errTest, err)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_CreateSecret_CreateSecretErr(t *testing.T) {
	data := map[string][]byte{"str": []byte("str")}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.Nexus{})
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	labels := platformHelper.GenerateLabels(instance.Name)

	secretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}
	err := controllerutil.SetControllerReference(&instance, secretObject, scheme)
	if err != nil {
		t.Fatal(err)
	}

	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}

	errTest := errors.New("test")
	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))
	secrets.On("Create", context.TODO(), secretObject, metav1.CreateOptions{}).Return(nil, errTest)

	service := K8SService{
		Scheme:     scheme,
		CoreClient: &coreClient,
	}
	err = service.CreateSecret(instance, name, data)
	assert.Equal(t, errTest, err)
	coreClient.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestK8SService_CreateSecret(t *testing.T) {
	data := map[string][]byte{"str": []byte("str")}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.Nexus{})
	instance := v1alpha1.Nexus{ObjectMeta: createObjectMeta()}
	labels := platformHelper.GenerateLabels(instance.Name)

	secretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}
	err := controllerutil.SetControllerReference(&instance, secretObject, scheme)
	if err != nil {
		t.Fatal(err)
	}

	coreClient := kMock.CoreV1Interface{}
	secrets := &kMock.SecretInterface{}

	coreClient.On("Secrets", namespace).Return(secrets)
	secrets.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, name))
	secrets.On("Create", context.TODO(), secretObject, metav1.CreateOptions{}).Return(secretObject, nil)

	service := K8SService{
		Scheme:     scheme,
		CoreClient: &coreClient,
	}
	err = service.CreateSecret(instance, name, data)
	assert.NoError(t, err)
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
	err = service.Init(restConfig, scheme, client)
	assert.NoError(t, err)
}
