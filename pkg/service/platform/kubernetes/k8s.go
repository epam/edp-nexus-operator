package kubernetes

import (
	"context"
	"fmt"
	"strings"

	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	edpCompApi "github.com/epam/edp-component-operator/api/v1"
	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	nexusV1 "github.com/epam/edp-nexus-operator/v2/api/v1"
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
