package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	edpCompApi "github.com/epam/edp-component-operator/api/v1"
	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/api/v1"
	nexusV1 "github.com/epam/edp-nexus-operator/v2/api/v1"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
	platformHelper "github.com/epam/edp-nexus-operator/v2/pkg/service/platform/helper"
)

const (
	crHTTPSKey     = "https"
	formatHTTPS    = "%s://%s%s"
	crNamespaceKey = "Namespace"
	crNameKey      = "Name"
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

// K8SService struct for K8S platform service.
type K8SService struct {
	Scheme             *runtime.Scheme
	CoreClient         CoreClient
	client             client.Client
	appClient          K8SClient
	networkingV1Client NetworkingClient
}

func (s *K8SService) IsDeploymentReady(instance *nexusV1.Nexus) (*bool, error) {
	var res *bool

	dc, err := s.appClient.Deployments(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
	if err != nil {
		return res, fmt.Errorf("failed to get deployment %s: %w", instance.Name, err)
	}

	t := dc.Status.UpdatedReplicas == 1 && dc.Status.AvailableReplicas == 1
	res = &t

	return res, nil
}

func (s *K8SService) AddKeycloakProxyToDeployConf(instance *nexusV1.Nexus, args []string) error {
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
		return fmt.Errorf("failed to get deployment %s: %w", instance.Name, err)
	}

	if platformHelper.ContainerInDeployConf(old.Spec.Template.Spec.Containers, &c) {
		log.V(1).Info("Keycloak proxy is present", crNamespaceKey, instance.Namespace, crNameKey, instance.Name)

		return nil
	}

	old.Spec.Template.Spec.Containers = append(old.Spec.Template.Spec.Containers, c)

	if _, err = s.appClient.Deployments(instance.Namespace).Update(context.TODO(), old, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to get deployment %s: %w", instance.Name, err)
	}

	log.Info("Keycloak proxy added.", crNamespaceKey, instance.Namespace, crNameKey, instance.Name)

	return nil
}

func (s *K8SService) GetExternalUrl(namespace, name string) (url, host, schema string, err error) {
	i, err := s.networkingV1Client.Ingresses(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Ingress not found", crNamespaceKey, namespace, crNameKey, name)
			return "", "", "", nil
		}

		return "", "", "", fmt.Errorf("failed to get ingress: %w", err)
	}

	host = i.Spec.Rules[0].Host
	port := strings.TrimRight(i.Spec.Rules[0].HTTP.Paths[0].Path, platformHelper.UrlCutset)

	return fmt.Sprintf(formatHTTPS, crHTTPSKey, host, port), host, crHTTPSKey, nil
}

func (s *K8SService) UpdateExternalTargetPath(instance *nexusV1.Nexus, targetPort intstr.IntOrString) error {
	i, err := s.GetIngressByCr(instance)
	if err != nil {
		return fmt.Errorf("failed to get route: %w", err)
	}

	if i.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number == int32(targetPort.IntValue()) {
		log.V(1).Info(
			"Target Port is already set",
			crNamespaceKey,
			instance.Namespace,
			crNameKey,
			instance.Name,
			"TargetPort",
			targetPort.StrVal,
			"IngressName",
			i.Name,
		)

		return nil
	}

	i.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number = int32(targetPort.IntValue())

	if _, err = s.networkingV1Client.Ingresses(instance.Namespace).Update(context.TODO(), i, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update ingress: %w", err)
	}

	return nil
}

// Init initializes *K8SService.
func (s *K8SService) Init(c *rest.Config, scheme *runtime.Scheme, k8sClient client.Client) error {
	CoreClient, err := coreV1Client.NewForConfig(c)
	if err != nil {
		return fmt.Errorf("failed to initialize coreV1 client: %w", err)
	}

	ac, err := appsV1Client.NewForConfig(c)
	if err != nil {
		return fmt.Errorf("failed to initialize appsV1 client: %w", err)
	}

	ec, err := networkingV1Client.NewForConfig(c)
	if err != nil {
		return fmt.Errorf("failed to initialize networkingV1 client: %w", err)
	}

	s.CoreClient = CoreClient
	s.client = k8sClient
	s.Scheme = scheme
	s.appClient = ac
	s.networkingV1Client = ec

	return nil
}

// CreateSecret creates secret object in K8s cluster.
func (s *K8SService) CreateSecret(instance *nexusV1.Nexus, name string, data map[string][]byte) error {
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

	if err := controllerutil.SetControllerReference(instance, secretObject, s.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	_, err := s.CoreClient.Secrets(secretObject.Namespace).Get(context.TODO(), secretObject.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to get secrets: %w", err)
	}

	secret, err := s.CoreClient.Secrets(secretObject.Namespace).Create(context.TODO(), secretObject, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secrets: %w", err)
	}

	log.Info("Secret has been created", crNamespaceKey, instance.Namespace, crNameKey, instance.Name, "SecretName", secret.Name)

	return nil
}

// GetServiceByCr return Service object by name.
func (s *K8SService) GetServiceByCr(name, namespace string) (*coreV1Api.Service, error) {
	service, err := s.CoreClient.Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("service %s not found", name)
		}

		return nil, fmt.Errorf("failed to get service - %s: %w", name, err)
	}

	return service, nil
}

// AddPortToService performs adding new port in Service in K8S.
func (s *K8SService) AddPortToService(instance *nexusV1.Nexus, newPortSpec *coreV1Api.ServicePort) error {
	svc, err := s.GetServiceByCr(instance.Name, instance.Namespace)
	if err != nil || svc == nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	if platformHelper.PortInService(svc.Spec.Ports, newPortSpec) {
		log.V(1).Info(
			"Port is already in s",
			crNamespaceKey,
			instance.Namespace,
			crNameKey,
			instance.Name,
			"Port",
			newPortSpec.Name,
			"ServiceName",
			svc.Name,
		)

		return nil
	}

	svc.Spec.Ports = append(svc.Spec.Ports, *newPortSpec)

	if _, err = s.CoreClient.Services(instance.Namespace).Update(context.TODO(), svc, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update k8s service: %w", err)
	}

	return nil
}

// CreateConfigMapFromFile performs creating ConfigMap in K8S.
func (s *K8SService) CreateConfigMapFromFile(instance *nexusV1.Nexus, configMapName, path string) error {
	configMapData := make(map[string]string)

	pathInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to open path %s: %w", path, err)
	}

	if pathInfo.Mode().IsDir() {
		directory, rErr := os.ReadDir(path)
		if rErr != nil {
			return fmt.Errorf("failed to open path %s: %w", path, rErr)
		}

		for _, file := range directory {
			content, readErr := os.ReadFile(fmt.Sprintf("%s/%s", path, file.Name()))
			if readErr != nil {
				return fmt.Errorf("failed to open path %s: %w", path, err)
			}

			configMapData[file.Name()] = string(content)
		}
	} else {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read file %s", path)
		}

		configMapData = map[string]string{
			filepath.Base(path): string(content),
		}
	}

	labels := platformHelper.GenerateLabels(instance.Name)
	configMapObject := &coreV1Api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: instance.Namespace, Labels: labels},
		Data:       configMapData,
	}

	if err = controllerutil.SetControllerReference(instance, configMapObject, s.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if _, err = s.CoreClient.ConfigMaps(instance.Namespace).Get(context.TODO(), configMapObject.Name, metav1.GetOptions{}); err == nil {
		return nil
	}

	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to get config map - %s: %w", configMapObject.Name, err)
	}

	cm, err := s.CoreClient.ConfigMaps(configMapObject.Namespace).Create(context.TODO(), configMapObject, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create config map - %s: %w", configMapObject.Name, err)
	}

	log.Info("ConfigMap has been created", crNamespaceKey, instance.Namespace, crNameKey, instance.Name, "ConfigMapName", cm.Name)

	return nil
}

// GetConfigMapData return data field of ConfigMap.
func (s *K8SService) GetConfigMapData(namespace, name string) (map[string]string, error) {
	configMap, err := s.CoreClient.ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		log.Error(err, "config map not found", crNamespaceKey, namespace, crNameKey, name, "ConfigMapName", name)
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get config map - %s: %w", name, err)
	}

	return configMap.Data, nil
}

// GetSecretData return data field of Secret.
func (s *K8SService) GetSecretData(namespace, name string) (map[string][]byte, error) {
	secret, err := s.CoreClient.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		log.Error(err, "secret not found", crNamespaceKey, namespace, crNameKey, name, "SecretName", name)

		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	}

	return secret.Data, nil
}

func (s *K8SService) GetSecret(namespace, name string) (*coreV1Api.Secret, error) {
	coreSecret, err := s.CoreClient.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return coreSecret, nil
}

func (s *K8SService) UpdateSecret(secret *coreV1Api.Secret) error {
	if _, err := s.CoreClient.Secrets(secret.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

func (s *K8SService) CreateJenkinsServiceAccount(namespace, secretName string) error {
	_, err := s.getJenkinsServiceAccount(secretName, namespace)
	if err == nil {
		return nil
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

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

	if err = s.client.Create(context.TODO(), jsa); err != nil {
		return fmt.Errorf("failed to create jenkins service account: %w", err)
	}

	log.Info("JenkinsServiceAccount has been created", crNamespaceKey, namespace, "JenkinsServiceAccountName", jsa.Name)

	return nil
}

func (s *K8SService) getJenkinsServiceAccount(name, namespace string) (*jenkinsV1Api.JenkinsServiceAccount, error) {
	jsa := &jenkinsV1Api.JenkinsServiceAccount{}
	if err := s.client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, jsa); err != nil {
		return nil, fmt.Errorf("failed to get jenkins service account: %w", err)
	}

	return jsa, nil
}

func (s *K8SService) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	nsn := types.NamespacedName{
		Namespace: kc.Namespace,
		Name:      kc.Name,
	}

	err := s.client.Get(context.TODO(), nsn, kc)
	if err == nil {
		return nil
	}

	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to get keyclock client: %w", err)
	}

	if err = s.client.Create(context.TODO(), kc); err != nil {
		return fmt.Errorf("failed to create Keycloak client %s: %w", kc.Name, err)
	}

	log.Info("Keycloak client created", crNamespaceKey, kc.Namespace, crNameKey, kc.Name, "KeycloakClientName", kc.Name)

	return nil
}

func (s *K8SService) GetKeycloakClient(name, namespace string) (keycloakV1Api.KeycloakClient, error) {
	out := keycloakV1Api.KeycloakClient{}
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	err := s.client.Get(context.TODO(), nsn, &out)
	if err != nil {
		return out, fmt.Errorf("failed to get keyclock client: %w", err)
	}

	return out, nil
}

func (s *K8SService) GetIngressByCr(instance *nexusV1.Nexus) (*networkingV1.Ingress, error) {
	i, err := s.networkingV1Client.Ingresses(instance.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ingresses list from the cluster: %w", err)
	}

	for j := range i.Items {
		if i.Items[j].Name == instance.Name {
			return &i.Items[j], nil
		}
	}

	return nil, nil
}

func (s *K8SService) CreateEDPComponentIfNotExist(n *nexusV1.Nexus, url, icon string) error {
	if _, err := s.getEDPComponent(n.Name, n.Namespace); err != nil {
		if k8serrors.IsNotFound(err) {
			return s.createEDPComponent(n, url, icon)
		}

		return fmt.Errorf("failed to get edp component: %s: %w", n.Name, err)
	}

	log.Info("edp component already exists", crNameKey, n.Name)

	return nil
}

func (s *K8SService) getEDPComponent(name, namespace string) (*edpCompApi.EDPComponent, error) {
	c := &edpCompApi.EDPComponent{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, c); err != nil {
		return nil, fmt.Errorf("failed to get EDP component: %w", err)
	}

	return c, nil
}

func (s *K8SService) createEDPComponent(n *nexusV1.Nexus, url, icon string) error {
	obj := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.Name,
			Namespace: n.Namespace,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "nexus",
			Url:     url,
			Icon:    icon,
			Visible: true,
		},
	}
	if err := controllerutil.SetControllerReference(n, obj, s.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := s.client.Create(context.TODO(), obj); err != nil {
		return fmt.Errorf("failed to create k8s service: %w", err)
	}

	return nil
}
