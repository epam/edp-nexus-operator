package kubernetes

import (
	"errors"
	"fmt"
	"io/ioutil"
	coreV1Api "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"log"
	"nexus-operator/pkg/apis/edp/v1alpha1"
	"nexus-operator/pkg/helper"
	nexusDefaultSpec "nexus-operator/pkg/service/nexus/spec"
	platformHelper "nexus-operator/pkg/service/platform/helper"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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
			log.Printf("[INFO] PersistantVolumeClaim %s/%s has been created", volume.Namespace, volume.Name)
		} else if err != nil {
			return helper.LogErrorAndReturn(err)
		}
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
		log.Printf("[INFO] ServiceAccount %s/%s has been created", serviceAccount.Namespace, serviceAccount.Name)
	} else if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}

	return serviceAccount, nil
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
		log.Printf("[INFO] Service %s/%s has been created", svc.Namespace, svc.Name)
	} else if err != nil {
		return helper.LogErrorAndReturn(err)
	}

	return nil
}

// CreateConfigMapFromFile performs creating ConfigMap in K8S
func (service K8SService) CreateConfigMapFromFile(instance v1alpha1.Nexus, configMapName string, filePath string) error {
	labels := platformHelper.GenerateLabels(instance.Name)
	data, err := ioutil.ReadFile(filePath)

	configMapObject := &coreV1Api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			filepath.Base(filePath): string(data),
		},
	}

	if err := controllerutil.SetControllerReference(&instance, configMapObject, service.Scheme); err != nil {
		return helper.LogErrorAndReturn(err)
	}

	configMap, err := service.CoreClient.ConfigMaps(instance.Namespace).Get(configMapObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		configMap, err = service.CoreClient.ConfigMaps(configMapObject.Namespace).Create(configMapObject)
		if err != nil {
			return helper.LogErrorAndReturn(err)
		}
		log.Printf("[INFO] ConfigMap %s/%s has been created", configMap.Namespace, configMapName)
	} else if err != nil {
		return helper.LogErrorAndReturn(err)
	}

	return nil
}

// CreateConfigMapFromFile performs creating ConfigMap in K8S
func (service K8SService) CreateConfigMapsFromDirectory(instance v1alpha1.Nexus, directoryPath string) error {
	directory, err := ioutil.ReadDir(directoryPath)
	if err != nil {
		return helper.LogErrorAndReturn(errors.New(fmt.Sprintf("[ERROR] Couldn't read directory %v with scripts for %v/%v. Err - %v.", directoryPath, instance.Namespace, instance.Name, err)))
	}

	for _, file := range directory {
		configMapName := fmt.Sprintf("%v-%v", instance.Name, file.Name())
		service.CreateConfigMapFromFile(instance, configMapName, fmt.Sprintf("%v/%v", directoryPath, file.Name()))
		if err != nil {
			return helper.LogErrorAndReturn(errors.New(fmt.Sprintf("[ERROR] Couldn't create config-map %v in namespace %v. Err - %v.", configMapName, instance.Namespace, err)))
		}
	}
	return nil
}

func (service K8SService) GetConfigMapData(namespace string, name string) (map[string]string, error) {
	configMap, err := service.CoreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Config map %v in namespace %v not found", name, namespace)
		return nil, nil
	} else if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}
	return configMap.Data, nil
}
