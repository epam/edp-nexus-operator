package kubernetes

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1alpha1"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
	platformHelper "github.com/epam/edp-nexus-operator/v2/pkg/service/platform/helper"
)

type CoreClient interface {
	coreV1Client.CoreV1Interface
}

type K8SClient interface {
	appsV1Client.AppsV1Interface
}

type NetworkingClient interface {
	networkingV1Client.NetworkingV1Interface
}

var log = ctrl.Log.WithName("platform")

// K8SService struct for K8S platform service
type K8SService struct {
	Scheme             *runtime.Scheme
	CoreClient         CoreClient
	client             client.Client
	appClient          K8SClient
	networkingV1Client NetworkingClient
}

func (s K8SService) IsDeploymentReady(instance v1alpha1.Nexus) (res *bool, err error) {
	dc, err := s.appClient.Deployments(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
	if err != nil {
		return
	}

	t := dc.Status.UpdatedReplicas == 1 && dc.Status.AvailableReplicas == 1
	res = &t
	return
}

func (s K8SService) AddKeycloakProxyToDeployConf(instance v1alpha1.Nexus, args []string) error {
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
		Args:                     args,
	}

	old, err := s.appClient.Deployments(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if platformHelper.ContainerInDeployConf(old.Spec.Template.Spec.Containers, c) {
		log.V(1).Info("Keycloak proxy is present", "Namespace", instance.Namespace, "Name", instance.Name)
		return nil
	}
	old.Spec.Template.Spec.Containers = append(old.Spec.Template.Spec.Containers, c)

	_, err = s.appClient.Deployments(instance.Namespace).Update(context.TODO(), old, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	log.Info("Keycloak proxy added.", "Namespace", instance.Namespace, "Name", instance.Name)
	return nil
}

func (s K8SService) GetExternalUrl(namespace string, name string) (webURL string, host string, scheme string, err error) {
	i, err := s.networkingV1Client.Ingresses(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Ingress not found", "Namespace", namespace, "Name", name)
			return "", "", "", nil
		}
		return "", "", "", err
	}

	h := i.Spec.Rules[0].Host
	sc := "https"
	p := strings.TrimRight(i.Spec.Rules[0].HTTP.Paths[0].Path, platformHelper.UrlCutset)

	return fmt.Sprintf("%s://%s%s", sc, h, p), h, sc, nil
}

func (s K8SService) UpdateExternalTargetPath(instance v1alpha1.Nexus, targetPort intstr.IntOrString) error {
	i, err := s.GetIngressByCr(instance)
	if err != nil {
		return errors.Wrap(err, "couldn't get route")
	}

	if i.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number == int32(targetPort.IntValue()) {
		log.V(1).Info("Target Port is already set",
			"Namespace", instance.Namespace, "Name", instance.Name, "TargetPort", targetPort.StrVal, "IngressName", i.Name)
		return nil
	}

	i.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number = int32(targetPort.IntValue())

	_, err = s.networkingV1Client.Ingresses(instance.Namespace).Update(context.TODO(), i, metav1.UpdateOptions{})
	return err
}

// Init initializes K8SService
func (s *K8SService) Init(c *rest.Config, Scheme *runtime.Scheme, k8sClient client.Client) error {
	CoreClient, err := coreV1Client.NewForConfig(c)
	if err != nil {
		return errors.Wrap(err, "coreV1 client initialization failed")
	}

	ac, err := appsV1Client.NewForConfig(c)
	if err != nil {
		return errors.New("appsV1 client initialization failed")
	}

	ec, err := networkingV1Client.NewForConfig(c)
	if err != nil {
		return errors.New("networkingV1 client initialization failed")
	}

	s.CoreClient = CoreClient
	s.client = k8sClient
	s.Scheme = Scheme
	s.appClient = ac
	s.networkingV1Client = ec
	return nil
}

//CreateSecret creates secret object in K8s cluster
func (s K8SService) CreateSecret(instance v1alpha1.Nexus, name string, data map[string][]byte) error {
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

	if err := controllerutil.SetControllerReference(&instance, secretObject, s.Scheme); err != nil {
		return err
	}

	_, err := s.CoreClient.Secrets(secretObject.Namespace).Get(context.TODO(), secretObject.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	secret, err := s.CoreClient.Secrets(secretObject.Namespace).Create(context.TODO(), secretObject, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Info("Secret has been created", "Namespace", instance.Namespace, "Name", instance.Name, "SecretName", secret.Name)

	return nil
}

// GetServiceByCr return Service object by name
func (s K8SService) GetServiceByCr(name, namespace string) (*coreV1Api.Service, error) {
	service, err := s.CoreClient.Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "couldn't find service by %v name", name)
		}
		return nil, err
	}
	return service, nil
}

// AddPortToService performs adding new port in Service in K8S
func (s K8SService) AddPortToService(instance v1alpha1.Nexus, newPortSpec coreV1Api.ServicePort) error {
	svc, err := s.GetServiceByCr(instance.Name, instance.Namespace)
	if err != nil || svc == nil {
		return errors.Wrap(err, "couldn't get s")
	}

	if platformHelper.PortInService(svc.Spec.Ports, newPortSpec) {
		log.V(1).Info("Port is already in s",
			"Namespace", instance.Namespace, "Name", instance.Name, "Port", newPortSpec.Name, "ServiceName", svc.Name)
		return nil
	}

	svc.Spec.Ports = append(svc.Spec.Ports, newPortSpec)

	if _, err = s.CoreClient.Services(instance.Namespace).Update(context.TODO(), svc, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

// CreateConfigMapFromFile performs creating ConfigMap in K8S
func (s K8SService) CreateConfigMapFromFile(instance v1alpha1.Nexus, configMapName string, path string) error {
	configMapData := make(map[string]string)
	pathInfo, err := os.Stat(path)
	if err != nil {
		return errors.Wrapf(err, "couldn't open path %v", path)
	}
	if pathInfo.Mode().IsDir() {
		directory, err := ioutil.ReadDir(path)
		if err != nil {
			return errors.Wrapf(err, "couldn't open path %v", path)
		}
		for _, file := range directory {
			content, err := ioutil.ReadFile(fmt.Sprintf("%v/%v", path, file.Name()))
			if err != nil {
				return errors.Wrapf(err, "couldn't open path %v", path)
			}
			configMapData[file.Name()] = string(content)
		}
	} else {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "couldn't read file %v", path)
		}
		configMapData = map[string]string{
			filepath.Base(path): string(content),
		}
	}

	labels := platformHelper.GenerateLabels(instance.Name)
	configMapObject := &coreV1Api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: configMapData,
	}

	if err := controllerutil.SetControllerReference(&instance, configMapObject, s.Scheme); err != nil {
		return err
	}

	_, err = s.CoreClient.ConfigMaps(instance.Namespace).Get(context.TODO(), configMapObject.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	cm, err := s.CoreClient.ConfigMaps(configMapObject.Namespace).Create(context.TODO(), configMapObject, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Info("ConfigMap has been created",
		"Namespace", instance.Namespace, "Name", instance.Name, "ConfigMapName", cm.Name)

	return nil
}

// GetConfigMapData return data field of ConfigMap
func (s K8SService) GetConfigMapData(namespace string, name string) (map[string]string, error) {
	configMap, err := s.CoreClient.ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		log.Error(err, "config map not found",
			"Namespace", namespace, "Name", name, "ConfigMapName", name)
		return nil, nil
	}

	return configMap.Data, err
}

// GetSecret return data field of Secret
func (s K8SService) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	secret, err := s.CoreClient.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		log.Error(err, "secret not found",
			"Namespace", namespace, "Name", name, "SecretName", name)
		return nil, nil
	}

	return secret.Data, err
}

func (s K8SService) GetSecret(namespace string, name string) (*coreV1Api.Secret, error) {
	return s.CoreClient.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (s K8SService) UpdateSecret(secret *coreV1Api.Secret) error {
	_, err := s.CoreClient.Secrets(secret.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	return err
}

func (s K8SService) CreateJenkinsServiceAccount(namespace string, secretName string) error {
	if _, err := s.getJenkinsServiceAccount(secretName, namespace); err != nil {
		if k8serrors.IsNotFound(err) {
			jsa := &jenkinsV1Api.JenkinsServiceAccount{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Spec: jenkinsV1Api.JenkinsServiceAccountSpec{
					Type:        "password",
					Credentials: secretName,
				},
			}

			if err := s.client.Create(context.TODO(), jsa); err != nil {
				return err
			}
			log.Info("JenkinsServiceAccount has been created", "Namespace", namespace, "JenkinsServiceAccountName", jsa.Name)
			return nil
		}
		return err
	}
	return nil
}

func (s K8SService) getJenkinsServiceAccount(name, namespace string) (*jenkinsV1Api.JenkinsServiceAccount, error) {
	jsa := &jenkinsV1Api.JenkinsServiceAccount{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, jsa)
	if err != nil {
		return nil, err
	}
	return jsa, nil
}

func (s K8SService) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	nsn := types.NamespacedName{
		Namespace: kc.Namespace,
		Name:      kc.Name,
	}

	err := s.client.Get(context.TODO(), nsn, kc)
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	err = s.client.Create(context.TODO(), kc)
	if err != nil {
		return errors.Wrapf(err, "failed to create Keycloak client %s", kc.Name)
	}
	log.Info("Keycloak client created",
		"Namespace", kc.Namespace, "Name", kc.Name, "KeycloakClientName", kc.Name)

	return nil
}

func (s K8SService) GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error) {
	out := keycloakV1Api.KeycloakClient{}
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	err := s.client.Get(context.TODO(), nsn, &out)
	if err != nil {
		return out, err
	}

	return out, nil
}

func (s K8SService) GetIngressByCr(instance v1alpha1.Nexus) (*networkingV1.Ingress, error) {
	i, err := s.networkingV1Client.Ingresses(instance.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve ingresses list from the cluster")
	}
	for _, e := range i.Items {
		if e.Name == instance.Name {
			return &e, nil
		}
	}
	return nil, nil
}

func (s K8SService) CreateEDPComponentIfNotExist(nexus v1alpha1.Nexus, url string, icon string) error {
	if _, err := s.getEDPComponent(nexus.Name, nexus.Namespace); err != nil {
		if k8serrors.IsNotFound(err) {
			return s.createEDPComponent(nexus, url, icon)
		}
		return errors.Wrapf(err, "failed to get edp component: %v", nexus.Name)
	}
	log.Info("edp component already exists", "name", nexus.Name)
	return nil
}

func (s K8SService) getEDPComponent(name, namespace string) (*edpCompApi.EDPComponent, error) {
	c := &edpCompApi.EDPComponent{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s K8SService) createEDPComponent(nexus v1alpha1.Nexus, url string, icon string) error {
	obj := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nexus.Name,
			Namespace: nexus.Namespace,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "nexus",
			Url:     url,
			Icon:    icon,
			Visible: true,
		},
	}
	if err := controllerutil.SetControllerReference(&nexus, obj, s.Scheme); err != nil {
		return err
	}
	return s.client.Create(context.TODO(), obj)
}
