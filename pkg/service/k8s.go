package service

import (
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
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type K8SService struct {
	scheme     *runtime.Scheme
	coreClient coreV1Client.CoreV1Client
}

func (service *K8SService) Init(config *rest.Config, scheme *runtime.Scheme) error {

	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}
	service.coreClient = *coreClient
	service.scheme = scheme
	return nil
}

func (service K8SService) CreateVolume(instance v1alpha1.Nexus) error {
	labels := generateLabels(instance.Name)

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

		if err := controllerutil.SetControllerReference(&instance, volumeObject, service.scheme); err != nil {
			return logErrorAndReturn(err)
		}

		volume, err := service.coreClient.PersistentVolumeClaims(volumeObject.Namespace).Get(volumeObject.Name, metav1.GetOptions{})

		if err != nil && k8serr.IsNotFound(err) {
			volume, err = service.coreClient.PersistentVolumeClaims(volumeObject.Namespace).Create(volumeObject)
			if err != nil {
				return logErrorAndReturn(err)
			}
			log.Printf("[INFO] PersistantVolumeClaim %s/%s has been created", volume.Namespace, volume.Name)
		} else if err != nil {
			return logErrorAndReturn(err)
		}
	}
	return nil
}

func (service K8SService) CreateServiceAccount(instance v1alpha1.Nexus) (*coreV1Api.ServiceAccount, error) {
	labels := generateLabels(instance.Name)

	serviceAccountObject := &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
	}

	if err := controllerutil.SetControllerReference(&instance, serviceAccountObject, service.scheme); err != nil {
		return nil, logErrorAndReturn(err)
	}

	serviceAccount, err := service.coreClient.ServiceAccounts(serviceAccountObject.Namespace).Get(serviceAccountObject.Name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		serviceAccount, err = service.coreClient.ServiceAccounts(serviceAccountObject.Namespace).Create(serviceAccountObject)
		if err != nil {
			return nil, logErrorAndReturn(err)
		}
		log.Printf("[INFO] ServiceAccount %s/%s has been created", serviceAccount.Namespace, serviceAccount.Name)
	} else if err != nil {
		return nil, logErrorAndReturn(err)
	}

	return serviceAccount, nil
}

func (service K8SService) CreateService(instance v1alpha1.Nexus) error {
	labels := generateLabels(instance.Name)

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
					Port:       NexusPort,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, serviceObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	svc, err := service.coreClient.Services(instance.Namespace).Get(serviceObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		svc, err = service.coreClient.Services(serviceObject.Namespace).Create(serviceObject)
		if err != nil {
			return logErrorAndReturn(err)
		}
		log.Printf("[INFO] Service %s/%s has been created", svc.Namespace, svc.Name)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service K8SService) CreateConfigMapFromFile(instance v1alpha1.Nexus, configMapName string, filePath string) error {
	labels := generateLabels(instance.Name)
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

	if err := controllerutil.SetControllerReference(&instance, configMapObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	configMap, err := service.coreClient.ConfigMaps(instance.Namespace).Get(configMapObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		configMap, err = service.coreClient.ConfigMaps(configMapObject.Namespace).Create(configMapObject)
		if err != nil {
			return logErrorAndReturn(err)
		}
		log.Printf("[INFO] ConfigMap %s/%s has been created", configMap.Namespace, configMapName)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}
