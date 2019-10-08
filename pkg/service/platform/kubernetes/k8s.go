package kubernetes

import (
	"context"
	"fmt"
	jenkinsV1Api "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsV1Client "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkinsserviceaccount/client"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	//_ "github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"github.com/epmd-edp/nexus-operator/v2/pkg/apis/edp/v1alpha1"
	nexusDefaultSpec "github.com/epmd-edp/nexus-operator/v2/pkg/service/nexus/spec"
	platformHelper "github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/helper"
	"github.com/pkg/errors"
	"io/ioutil"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// K8SService struct for K8S platform service
type K8SService struct {
	Scheme                      *runtime.Scheme
	CoreClient                  coreV1Client.CoreV1Client
	JenkinsServiceAccountClient jenkinsV1Client.EdpV1Client
	k8sUnstructuredClient       client.Client
}

func (s K8SService) IsDeploymentReady(instance v1alpha1.Nexus) (*bool, error) {
	t := false
	return &t, nil
}

func (s K8SService) AddKeycloakProxyToDeployConf(instance v1alpha1.Nexus, keycloakClientConf []string) error {
	return nil
}

func (s K8SService) GetExternalUrl(namespace string, name string) (webURL string, scheme string, err error) {
	return "","",nil
}

func (s K8SService) UpdateRouteTarget(instance v1alpha1.Nexus, targetPort intstr.IntOrString) error {
	return nil
}


func (s K8SService) CreateDeployConf(instance v1alpha1.Nexus) error {
	return nil
}

func (s K8SService) CreateExternalEndpoint(instance v1alpha1.Nexus) error {
	return nil
}

// Init initializes K8SService
func (s *K8SService) Init(config *rest.Config, Scheme *runtime.Scheme, k8sClient *client.Client) error {
	CoreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize Core V1 Client")
	}

	JenkinsServiceAccountClient, err := jenkinsV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize Jenkins Service Account Client")
	}

	s.CoreClient = *CoreClient
	s.JenkinsServiceAccountClient = *JenkinsServiceAccountClient
	s.k8sUnstructuredClient = *k8sClient
	s.Scheme = Scheme
	return nil
}

// CreateVolume performs creating PersistentVolumeClaim in K8S
func (s K8SService) CreateVolume(instance v1alpha1.Nexus) error {
	labels := platformHelper.GenerateLabels(instance.Name)

	for _, volume := range instance.Spec.Volumes {
		volumeObject := &coreV1Api.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-" + volume.Name,
				Namespace: instance.Namespace,
				Labels:    labels,
			},
			Spec: coreV1Api.PersistentVolumeClaimSpec{
				AccessModes: []coreV1Api.PersistentVolumeAccessMode{
					coreV1Api.ReadWriteOnce,
				},
				StorageClassName: &volume.StorageClass,
				Resources: coreV1Api.ResourceRequirements{
					Requests: map[coreV1Api.ResourceName]resource.Quantity{
						coreV1Api.ResourceStorage: resource.MustParse(volume.Capacity),
					},
				},
			},
		}

		if err := controllerutil.SetControllerReference(&instance, volumeObject, s.Scheme); err != nil {
			return err
		}

		volume, err := s.CoreClient.PersistentVolumeClaims(volumeObject.Namespace).Get(volumeObject.Name, metav1.GetOptions{})
		if err == nil {
			return err
		}

		if !k8serrors.IsNotFound(err) {
			return err
		}
		volume, err = s.CoreClient.PersistentVolumeClaims(volumeObject.Namespace).Create(volumeObject)
		if err != nil {
			return err
		}

		log.Info("Volume has been created", "Namespace", instance.Namespace, "Name", instance.Name, "VolumeName", volume.Name)
	}
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

	secret, err := s.CoreClient.Secrets(secretObject.Namespace).Get(secretObject.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	secret, err = s.CoreClient.Secrets(secretObject.Namespace).Create(secretObject)
	if err != nil {
		return err
	}
	log.Info("Secret has been created", "Namespace", instance.Namespace, "Name", instance.Name, "SecretName", secret.Name)

	return nil
}

// CreateServiceAccount performs creating ServiceAccount in K8S
func (s K8SService) CreateServiceAccount(instance v1alpha1.Nexus) error {
	labels := platformHelper.GenerateLabels(instance.Name)

	svcAccObj := &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
	}

	if err := controllerutil.SetControllerReference(&instance, svcAccObj, s.Scheme); err != nil {
		return err
	}

	svcAcc, err := s.CoreClient.ServiceAccounts(svcAccObj.Namespace).Get(svcAccObj.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	svcAcc, err = s.CoreClient.ServiceAccounts(svcAccObj.Namespace).Create(svcAccObj)
	if err != nil {
		return err
	}
	log.Info("ServiceAccount has been created", "Namespace", instance.Namespace, "Name", instance.Name, "ServiceAccountName", svcAcc.Name)

	return nil
}

// GetServiceByCr return Service object with instance as a reference owner
func (s K8SService) GetServiceByCr(instance v1alpha1.Nexus) (*coreV1Api.Service, error) {
	serviceList, err := s.CoreClient.Services(instance.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve services list from the cluster")
	}
	for _, service := range serviceList.Items {
		for _, owner := range service.OwnerReferences {
			if owner.UID == instance.UID {
				return &service, nil
			}
		}
	}
	return nil, nil
}

// AddPortToService performs adding new port in Service in K8S
func (s K8SService) AddPortToService(instance v1alpha1.Nexus, newPortSpec coreV1Api.ServicePort) error {
	svc, err := s.GetServiceByCr(instance)
	if err != nil || svc == nil {
		return errors.Wrap(err, "couldn't get s")
	}

	if platformHelper.PortInService(svc.Spec.Ports, newPortSpec) {
		log.V(1).Info("Port is already in s",
			"Namespace", instance.Namespace, "Name", instance.Name, "Port", newPortSpec.Name, "ServiceName", svc.Name)
		return nil
	}

	svc.Spec.Ports = append(svc.Spec.Ports, newPortSpec)

	if _, err = s.CoreClient.Services(instance.Namespace).Update(svc); err != nil {
		return err
	}
	return nil
}

// CreateService performs creating Service in K8S
func (s K8SService) CreateService(instance v1alpha1.Nexus) error {
	labels := platformHelper.GenerateLabels(instance.Name)

	serviceObject := &coreV1Api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: coreV1Api.ServiceSpec{
			Selector: labels,
			Ports: []coreV1Api.ServicePort{
				{
					TargetPort: intstr.IntOrString{StrVal: instance.Name},
					Port:       nexusDefaultSpec.NexusPort,
					Name:       "nexus-http",
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, serviceObject, s.Scheme); err != nil {
		return err
	}

	svc, err := s.CoreClient.Services(instance.Namespace).Get(serviceObject.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	svc, err = s.CoreClient.Services(serviceObject.Namespace).Create(serviceObject)
	if err != nil {
		return err
	}
	log.Info("Service has been created",
		"Namespace", instance.Namespace, "Name", instance.Name, "ServiceName", svc.Name)

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

	cm, err := s.CoreClient.ConfigMaps(instance.Namespace).Get(configMapObject.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	cm, err = s.CoreClient.ConfigMaps(configMapObject.Namespace).Create(configMapObject)
	if err != nil {
		return err
	}
	log.Info("ConfigMap has been created",
		"Namespace", instance.Namespace, "Name", instance.Name, "ConfigMapName", cm.Name)

	return nil
}

// CreateConfigMapFromFile performs creating ConfigMap in K8S
func (s K8SService) CreateConfigMapsFromDirectory(instance v1alpha1.Nexus, directoryPath string, createDedicatedConfigMaps bool) error {
	directory, err := ioutil.ReadDir(directoryPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't read directory %v with scripts", directoryPath)
	}

	if !createDedicatedConfigMaps {
		configMapName := fmt.Sprintf("%v-%v", instance.Name, filepath.Base(directoryPath))
		err = s.CreateConfigMapFromFile(instance, configMapName, directoryPath)
		if err != nil {
			return errors.Wrapf(err, "couldn't create config-map %v", configMapName)
		}
		return nil
	}

	for _, file := range directory {
		configMapName := fmt.Sprintf("%v-%v", instance.Name, file.Name())
		err = s.CreateConfigMapFromFile(instance, configMapName, fmt.Sprintf("%v/%v", directoryPath, file.Name()))
		if err != nil {
			return errors.Wrapf(err, "couldn't create config-map %v", configMapName)
		}
	}
	return nil
}

// GetConfigMapData return data field of ConfigMap
func (s K8SService) GetConfigMapData(namespace string, name string) (map[string]string, error) {
	configMap, err := s.CoreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		log.Error(err, "config map not found",
			"Namespace", namespace, "Name", name, "ConfigMapName", name)
		return nil, nil
	}

	return configMap.Data, err
}

// GetSecret return data field of Secret
func (s K8SService) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	secret, err := s.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		log.Error(err, "secret not found",
			"Namespace", namespace, "Name", name, "SecretName", name)
		return nil, nil
	}

	return secret.Data, err
}

func (s K8SService) GetSecret(namespace string, name string) (*coreV1Api.Secret, error) {
	return s.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
}

func (s K8SService) UpdateSecret(secret *coreV1Api.Secret) error {
	_, err := s.CoreClient.Secrets(secret.Namespace).Update(secret)
	return err
}

func (s K8SService) CreateJenkinsServiceAccount(namespace string, secretName string) error {

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

	_, err := s.JenkinsServiceAccountClient.Get(secretName, namespace, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	_, err = s.JenkinsServiceAccountClient.Create(jsa, namespace)
	if err != nil {
		return err
	}

	log.Info("JenkinsServiceAccount has been created", "Namespace", namespace, "JenkinsServiceAccountName", jsa.Name)

	return nil
}

func (s K8SService) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	nsn := types.NamespacedName{
		Namespace: kc.Namespace,
		Name:      kc.Name,
	}

	err := s.k8sUnstructuredClient.Get(context.TODO(), nsn, kc)
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	err = s.k8sUnstructuredClient.Create(context.TODO(), kc)
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

	err := s.k8sUnstructuredClient.Get(context.TODO(), nsn, &out)
	if err != nil {
		return out, err
	}

	return out, nil
}
