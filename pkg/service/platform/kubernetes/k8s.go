package kubernetes

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	coreV1Api "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"nexus-operator/pkg/apis/edp/v1alpha1"
	"nexus-operator/pkg/helper"
	nexusDefaultSpec "nexus-operator/pkg/service/nexus/spec"
	platformHelper "nexus-operator/pkg/service/platform/helper"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// K8SService struct for K8S platform service
type K8SService struct {
	Scheme     *runtime.Scheme
	CoreClient coreV1Client.CoreV1Client
}

// Init initializes K8SService
func (service *K8SService) Init(config *rest.Config, Scheme *runtime.Scheme) error {
	CoreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return helper.LogErrorAndReturn(err)
	}
	service.CoreClient = *CoreClient
	service.Scheme = Scheme
	return nil
}

// CreateVolume performs creating PersistentVolumeClaim in K8S
func (service K8SService) CreateVolume(instance v1alpha1.Nexus) error {
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

		if err := controllerutil.SetControllerReference(&instance, volumeObject, service.Scheme); err != nil {
			return helper.LogErrorAndReturn(err)
		}

		volume, err := service.CoreClient.PersistentVolumeClaims(volumeObject.Namespace).Get(volumeObject.Name, metav1.GetOptions{})

		if err != nil && k8serr.IsNotFound(err) {
			volume, err = service.CoreClient.PersistentVolumeClaims(volumeObject.Namespace).Create(volumeObject)
			if err != nil {
				return helper.LogErrorAndReturn(err)
			}
			log.Info(fmt.Sprintf("PersistantVolumeClaim %s/%s has been created", volume.Namespace, volume.Name))
		} else if err != nil {
			return helper.LogErrorAndReturn(err)
		}
	}
	return nil
}

//CreateSecret creates secret object in K8s cluster
func (service K8SService) CreateSecret(instance v1alpha1.Nexus, name string, data map[string][]byte) error {

	labels := platformHelper.GenerateLabels(instance.Name)

	nexusSecretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}

	if err := controllerutil.SetControllerReference(&instance, nexusSecretObject, service.Scheme); err != nil {
		return err
	}

	nexusSecret, err := service.CoreClient.Secrets(nexusSecretObject.Namespace).Get(nexusSecretObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		nexusSecret, err = service.CoreClient.Secrets(nexusSecretObject.Namespace).Create(nexusSecretObject)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Secret %s/%s has been created", nexusSecret.Namespace, nexusSecret.Name))
	} else if err != nil {
		return err
	}

	return nil
}

// CreateServiceAccount performs creating ServiceAccount in K8S
func (service K8SService) CreateServiceAccount(instance v1alpha1.Nexus) (*coreV1Api.ServiceAccount, error) {
	labels := platformHelper.GenerateLabels(instance.Name)

	serviceAccountObject := &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
	}

	if err := controllerutil.SetControllerReference(&instance, serviceAccountObject, service.Scheme); err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}

	serviceAccount, err := service.CoreClient.ServiceAccounts(serviceAccountObject.Namespace).Get(serviceAccountObject.Name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		serviceAccount, err = service.CoreClient.ServiceAccounts(serviceAccountObject.Namespace).Create(serviceAccountObject)
		if err != nil {
			return nil, helper.LogErrorAndReturn(err)
		}
		log.Info(fmt.Sprintf("ServiceAccount %s/%s has been created", serviceAccount.Namespace, serviceAccount.Name))
	} else if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}

	return serviceAccount, nil
}

// GetServiceByCr return Service object with instance as a reference owner
func (service K8SService) GetServiceByCr(instance v1alpha1.Nexus) (*coreV1Api.Service, error) {
	serviceList, err := service.CoreClient.Services(instance.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't retrieve services list from the cluster")
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
func (service K8SService) AddPortToService(instance v1alpha1.Nexus, newPortSpec coreV1Api.ServicePort) error {
	instanceService, err := service.GetServiceByCr(instance)
	if err != nil || instanceService == nil {
		return errors.Wrapf(err, "Couldn't get service for instance %v", instance.Name)
	}

	if platformHelper.PortInService(instanceService.Spec.Ports, newPortSpec) {
		log.V(1).Info(fmt.Sprintf("Port %v is already in service %v", newPortSpec.Name, instanceService.Name))
		return nil
	}

	instanceService.Spec.Ports = append(instanceService.Spec.Ports, newPortSpec)

	if _, err = service.CoreClient.Services(instance.Namespace).Update(instanceService); err != nil {
		return err
	}
	return nil
}

// CreateService performs creating Service in K8S
func (service K8SService) CreateService(instance v1alpha1.Nexus) error {
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

	if err := controllerutil.SetControllerReference(&instance, serviceObject, service.Scheme); err != nil {
		return helper.LogErrorAndReturn(err)
	}

	svc, err := service.CoreClient.Services(instance.Namespace).Get(serviceObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		svc, err = service.CoreClient.Services(serviceObject.Namespace).Create(serviceObject)
		if err != nil {
			return helper.LogErrorAndReturn(err)
		}
		log.Info(fmt.Sprintf("Service %s/%s has been created", svc.Namespace, svc.Name))
	} else if err != nil {
		return helper.LogErrorAndReturn(err)
	}

	return nil
}

// CreateConfigMapFromFile performs creating ConfigMap in K8S
func (service K8SService) CreateConfigMapFromFile(instance v1alpha1.Nexus, configMapName string, path string) error {
	configMapData := make(map[string]string)
	pathInfo, err := os.Stat(path)
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf("Couldn't open path %v.", path))
	}
	if pathInfo.Mode().IsDir() {
		directory, err := ioutil.ReadDir(path)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Couldn't open path %v.", path))
		}
		for _, file := range directory {
			content, err := ioutil.ReadFile(fmt.Sprintf("%v/%v", path, file.Name()))
			if err != nil {
				return errors.Wrapf(err, fmt.Sprintf("Couldn't open path %v.", path))
			}
			configMapData[file.Name()] = string(content)
		}
	} else {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Couldn't read file %v.", path))
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

	if err := controllerutil.SetControllerReference(&instance, configMapObject, service.Scheme); err != nil {
		return helper.LogErrorAndReturn(err)
	}

	cm, err := service.CoreClient.ConfigMaps(instance.Namespace).Get(configMapObject.Name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		cm, err = service.CoreClient.ConfigMaps(configMapObject.Namespace).Create(configMapObject)
		if err != nil {
			return helper.LogErrorAndReturn(err)
		}
		log.Info(fmt.Sprintf("ConfigMap %s/%s has been created", cm.Namespace, cm.Name))
	} else if err != nil {
		return helper.LogErrorAndReturn(err)
	}

	return nil
}

// CreateConfigMapFromFile performs creating ConfigMap in K8S
func (service K8SService) CreateConfigMapsFromDirectory(instance v1alpha1.Nexus, directoryPath string, createDedicatedConfigMaps bool) error {
	directory, err := ioutil.ReadDir(directoryPath)
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf("Couldn't read directory %v with scripts for %v/%v.", directoryPath, instance.Namespace, instance.Name))
	}

	if !createDedicatedConfigMaps {
		configMapName := fmt.Sprintf("%v-%v", instance.Name, filepath.Base(directoryPath))
		err = service.CreateConfigMapFromFile(instance, configMapName, directoryPath)
		if err != nil {
			return errors.Wrapf(err, "Couldn't create config-map %v in namespace %v.", configMapName, instance.Namespace)
		}
		return nil
	}

	for _, file := range directory {
		configMapName := fmt.Sprintf("%v-%v", instance.Name, file.Name())
		err = service.CreateConfigMapFromFile(instance, configMapName, fmt.Sprintf("%v/%v", directoryPath, file.Name()))
		if err != nil {
			return errors.Wrapf(err, "Couldn't create config-map %v in namespace %v.", configMapName, instance.Namespace)
		}
	}
	return nil
}

// GetConfigMapData return data field of ConfigMap
func (service K8SService) GetConfigMapData(namespace string, name string) (map[string]string, error) {
	configMap, err := service.CoreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		log.Error(err, fmt.Sprintf("Config map %v in namespace %v not found", name, namespace))
		return nil, nil
	} else if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}
	return configMap.Data, nil
}

// GetSecret return data field of Secret
func (service K8SService) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	secret, err := service.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		log.Error(err, fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (service K8SService) GetSecret(namespace string, name string) (*coreV1Api.Secret, error) {
	secret, err := service.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		return nil, err
	} else if err != nil {
		return nil, err
	}
	return secret, nil
}

func (service K8SService) UpdateSecret(secret *coreV1Api.Secret) error {

	_, err := service.CoreClient.Secrets(secret.Namespace).Update(secret)
	if err != nil {
		return err
	}

	return nil
}
