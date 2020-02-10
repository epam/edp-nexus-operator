package kubernetes

import (
	"context"
	"fmt"
	edpCompApi "github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	edpCompClient "github.com/epmd-edp/edp-component-operator/pkg/client"
	jenkinsV1Api "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsV1Client "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkinsserviceaccount/client"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/nexus-operator/v2/pkg/apis/edp/v1alpha1"
	nexusDefaultSpec "github.com/epmd-edp/nexus-operator/v2/pkg/service/nexus/spec"
	platformHelper "github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/helper"
	"github.com/pkg/errors"
	"io/ioutil"
	appsV1Api "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	extensionsV1Api "k8s.io/api/extensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	extensionsV1Client "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
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
	appClient                   appsV1Client.AppsV1Client
	extensionsV1Client          extensionsV1Client.ExtensionsV1beta1Client
	edpCompClient               edpCompClient.EDPComponentV1Client
}

func (s K8SService) IsDeploymentReady(instance v1alpha1.Nexus) (res *bool, err error) {
	dc, err := s.appClient.Deployments(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
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
		Image:           nexusDefaultSpec.NexusKeycloakProxyImage,
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

	old, err := s.appClient.Deployments(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if platformHelper.ContainerInDeployConf(old.Spec.Template.Spec.Containers, c) {
		log.V(1).Info("Keycloak proxy is present", "Namespace", instance.Namespace, "Name", instance.Name)
		return nil
	}
	old.Spec.Template.Spec.Containers = append(old.Spec.Template.Spec.Containers, c)

	_, err = s.appClient.Deployments(instance.Namespace).Update(old)
	if err != nil {
		return err
	}

	log.Info("Keycloak proxy added.", "Namespace", instance.Namespace, "Name", instance.Name)
	return nil
}

func (s K8SService) GetExternalUrl(namespace string, name string) (webURL string, scheme string, err error) {
	i, err := s.extensionsV1Client.Ingresses(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Ingress not found", "Namespace", namespace, "Name", name)
			return "", "", nil
		}
		return "", "", err
	}

	h := i.Spec.Rules[0].Host
	sc := "https"

	return fmt.Sprintf("%s://%s", sc, h), sc, nil
}

func (s K8SService) UpdateExternalTargetPath(instance v1alpha1.Nexus, targetPort intstr.IntOrString) error {
	i, err := s.GetIngressByCr(instance)
	if err != nil {
		return errors.Wrap(err, "couldn't get route")
	}

	if i.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort == targetPort {
		log.V(1).Info("Target Port is already set",
			"Namespace", instance.Namespace, "Name", instance.Name, "TargetPort", targetPort.StrVal, "IngressName", i.Name)
		return nil
	}

	i.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort = targetPort

	_, err = s.extensionsV1Client.Ingresses(instance.Namespace).Update(i)
	return err
}

func (s K8SService) CreateDeployment(instance v1alpha1.Nexus) error {
	l := platformHelper.GenerateLabels(instance.Name)
	var rc int32 = 1
	var fsg int64 = 200
	t := true
	f := false
	do := &appsV1Api.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    l,
		},
		Spec: appsV1Api.DeploymentSpec{
			Replicas: &rc,
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: l,
			},
			Template: coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: l,
				},
				Spec: coreV1Api.PodSpec{
					Containers: []coreV1Api.Container{
						{
							Name:            instance.Name,
							Image:           instance.Spec.Image + ":" + instance.Spec.Version,
							ImagePullPolicy: coreV1Api.PullAlways,
							Env: []coreV1Api.EnvVar{
								{
									Name:  "CONTEXT_PATH",
									Value: "/",
								},
							},
							Ports: []coreV1Api.ContainerPort{
								{
									ContainerPort: nexusDefaultSpec.NexusPort,
								},
							},
							LivenessProbe: &coreV1Api.Probe{
								FailureThreshold:    5,
								InitialDelaySeconds: 180,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								Handler: coreV1Api.Handler{
									TCPSocket: &coreV1Api.TCPSocketAction{
										Port: intstr.FromInt(nexusDefaultSpec.NexusPort),
									},
								},
							},
							ReadinessProbe: &coreV1Api.Probe{
								FailureThreshold:    3,
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								Handler: coreV1Api.Handler{
									TCPSocket: &coreV1Api.TCPSocketAction{
										Port: intstr.FromInt(nexusDefaultSpec.NexusPort),
									},
								},
							},
							TerminationMessagePath: "/dev/termination-log",
							Resources: coreV1Api.ResourceRequirements{
								Requests: map[coreV1Api.ResourceName]resource.Quantity{
									coreV1Api.ResourceMemory: resource.MustParse(nexusDefaultSpec.NexusMemoryRequest),
								},
							},
							SecurityContext: &coreV1Api.SecurityContext{
								AllowPrivilegeEscalation: &f,
							},
							VolumeMounts: []coreV1Api.VolumeMount{
								{
									MountPath: "/nexus-data",
									Name:      "data",
								},
								{
									MountPath: "/opt/sonatype/nexus/etc/nexus-default.properties",
									Name:      "config",
									SubPath:   "nexus-default.properties",
								},
							},
						},
					},
					SecurityContext: &coreV1Api.PodSecurityContext{
						FSGroup: &fsg,
						RunAsNonRoot: &t,
						RunAsUser: &fsg,
						RunAsGroup: &fsg,
					},
					ServiceAccountName: instance.Name,
					Volumes: []coreV1Api.Volume{
						{
							Name: "data",
							VolumeSource: coreV1Api.VolumeSource{
								PersistentVolumeClaim: &coreV1Api.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%v-data", instance.Name),
								},
							},
						},
						{
							Name: "config",
							VolumeSource: coreV1Api.VolumeSource{
								ConfigMap: &coreV1Api.ConfigMapVolumeSource{
									LocalObjectReference: coreV1Api.LocalObjectReference{Name: fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultPropertiesConfigMapPrefix)},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, do, s.Scheme); err != nil {
		return err
	}

	d, err := s.appClient.Deployments(do.Namespace).Get(do.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	d, err = s.appClient.Deployments(do.Namespace).Create(do)
	if err != nil {
		return err
	}

	log.Info("Deployment has been created",
		"Namespace", d.Name, "Name", d.Name, "DeploymentName", d.Name)

	return nil
}

func (s K8SService) CreateExternalEndpoint(instance v1alpha1.Nexus) error {
	l := platformHelper.GenerateLabels(instance.Name)

	cs, err := s.CoreClient.Services(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		log.Info("Nexus Service has not been found")
		return err
	}

	io := &extensionsV1Api.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    l,
		},
		Spec: extensionsV1Api.IngressSpec{
			Rules: []extensionsV1Api.IngressRule{
				{
					Host: fmt.Sprintf("%s-%s.%s", instance.Name, instance.Namespace, instance.Spec.EdpSpec.DnsWildcard),
					IngressRuleValue: extensionsV1Api.IngressRuleValue{
						HTTP: &extensionsV1Api.HTTPIngressRuleValue{
							Paths: []extensionsV1Api.HTTPIngressPath{
								{
									Path: "/",
									Backend: extensionsV1Api.IngressBackend{
										ServiceName: instance.Name,
										ServicePort: intstr.IntOrString{
											IntVal: cs.Spec.Ports[0].TargetPort.IntVal,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(&instance, io, s.Scheme); err != nil {
		return err
	}

	i, err := s.extensionsV1Client.Ingresses(io.Namespace).Get(io.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	i, err = s.extensionsV1Client.Ingresses(io.Namespace).Create(io)
	if err != nil {
		return err
	}

	log.Info("Ingress has been created",
		"Namespace", i.Namespace, "Name", i.Name, "IngressName", i.Name)

	return nil
}

// Init initializes K8SService
func (s *K8SService) Init(c *rest.Config, Scheme *runtime.Scheme, k8sClient *client.Client) error {
	CoreClient, err := coreV1Client.NewForConfig(c)
	if err != nil {
		return errors.Wrap(err, "coreV1 client initialization failed")
	}

	JenkinsServiceAccountClient, err := jenkinsV1Client.NewForConfig(c)
	if err != nil {
		return errors.Wrap(err, "jenkinsServiceAccountClientV1alpha client initialization failed")
	}

	ac, err := appsV1Client.NewForConfig(c)
	if err != nil {
		return errors.New("appsV1 client initialization failed")
	}

	ec, err := extensionsV1Client.NewForConfig(c)
	if err != nil {
		return errors.New("extensionsV1beta1 client initialization failed")
	}
	edpCl, err := edpCompClient.NewForConfig(c)
	if err != nil {
		return errors.Wrap(err, "failed to init edp component client")
	}
	s.CoreClient = *CoreClient
	s.JenkinsServiceAccountClient = *JenkinsServiceAccountClient
	s.k8sUnstructuredClient = *k8sClient
	s.Scheme = Scheme
	s.appClient = *ac
	s.extensionsV1Client = *ec
	s.edpCompClient = *edpCl
	return nil
}

func (s K8SService) CreateSecurityContext(instance v1alpha1.Nexus, priority int32) error {
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

func (s K8SService) GetIngressByCr(instance v1alpha1.Nexus) (*extensionsV1Api.Ingress, error) {
	i, err := s.extensionsV1Client.Ingresses(instance.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve ingresses list from the cluster")
	}
	for _, e := range i.Items {
		for _, owner := range e.OwnerReferences {
			if owner.UID == instance.UID {
				return &e, nil
			}
		}
	}
	return nil, nil
}

func (s K8SService) CreateEDPComponentIfNotExist(nexus v1alpha1.Nexus, url string, icon string) error {
	comp, err := s.edpCompClient.
		EDPComponents(nexus.Namespace).
		Get(nexus.Name, metav1.GetOptions{})
	if err == nil {
		log.Info("edp component already exists", "name", comp.Name)
		return nil
	}
	if k8serrors.IsNotFound(err) {
		return s.createEDPComponent(nexus, url, icon)
	}
	return errors.Wrapf(err, "failed to get edp component: %v", nexus.Name)
}

func (s K8SService) createEDPComponent(nexus v1alpha1.Nexus, url string, icon string) error {
	obj := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: nexus.Name,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type: "nexus",
			Url:  url,
			Icon: icon,
		},
	}
	if err := controllerutil.SetControllerReference(&nexus, obj, s.Scheme); err != nil {
		return err
	}
	_, err := s.edpCompClient.
		EDPComponents(nexus.Namespace).
		Create(obj)
	return err
}
