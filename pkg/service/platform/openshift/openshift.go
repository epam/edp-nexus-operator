package openshift

import (
	"fmt"
	appsV1Api "github.com/openshift/api/apps/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"log"
	"nexus-operator/pkg/apis/edp/v1alpha1"
	"nexus-operator/pkg/helper"
	nexusDefaultSpec "nexus-operator/pkg/service/nexus/spec"
	platformHelper "nexus-operator/pkg/service/platform/helper"
	"nexus-operator/pkg/service/platform/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// OpenshiftService struct for Openshift platform service
type OpenshiftService struct {
	kubernetes.K8SService

	appClient   appsV1client.AppsV1Client
	routeClient routeV1Client.RouteV1Client
}

// Init initializes OpenshiftService
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	err := service.K8SService.Init(config, scheme)
	if err != nil {
		return helper.LogErrorAndReturn(err)
	}

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return helper.LogErrorAndReturn(err)
	}
	service.appClient = *appClient

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return helper.LogErrorAndReturn(err)
	}
	service.routeClient = *routeClient

	return nil
}

// CreateDeployConf performs creating DeploymentConfig in Openshift
func (service OpenshiftService) CreateDeployConf(instance v1alpha1.Nexus) error {

	labels := platformHelper.GenerateLabels(instance.Name)
	deploymentConfigObject := &appsV1Api.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsV1Api.DeploymentConfigSpec{
			Replicas: 1,
			Triggers: []appsV1Api.DeploymentTriggerPolicy{
				{
					Type: appsV1Api.DeploymentTriggerOnConfigChange,
				},
			},
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.DeploymentStrategyTypeRolling,
			},
			Selector: labels,
			Template: &coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1Api.PodSpec{
					Containers: []coreV1Api.Container{
						{
							Name:            instance.Name,
							Image:           nexusDefaultSpec.NexusDockerImage + ":" + instance.Spec.Version,
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
									LocalObjectReference: coreV1Api.LocalObjectReference{Name: "nexus-properties"},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(&instance, deploymentConfigObject, service.Scheme); err != nil {
		return helper.LogErrorAndReturn(err)
	}

	deploymentConfig, err := service.appClient.DeploymentConfigs(deploymentConfigObject.Namespace).Get(deploymentConfigObject.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		deploymentConfig, err = service.appClient.DeploymentConfigs(deploymentConfigObject.Namespace).Create(deploymentConfigObject)
		if err != nil {
			return helper.LogErrorAndReturn(err)
		}

		log.Printf("[INFO] DeploymentConfig %s/%s has been created", deploymentConfig.Namespace, deploymentConfig.Name)
	} else if err != nil {
		return helper.LogErrorAndReturn(err)
	}

	return nil
}

// CreateExternalEndpoint performs creating Route in Openshift
func (service OpenshiftService) CreateExternalEndpoint(instance v1alpha1.Nexus) error {

	labels := platformHelper.GenerateLabels(instance.Name)

	routeObject := &routeV1Api.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: routeV1Api.RouteSpec{
			TLS: &routeV1Api.TLSConfig{
				Termination:                   routeV1Api.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routeV1Api.InsecureEdgeTerminationPolicyRedirect,
			},
			To: routeV1Api.RouteTargetReference{
				Name: instance.Name,
				Kind: "Service",
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, routeObject, service.Scheme); err != nil {
		return helper.LogErrorAndReturn(err)
	}

	route, err := service.routeClient.Routes(routeObject.Namespace).Get(routeObject.Name, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		route, err = service.routeClient.Routes(routeObject.Namespace).Create(routeObject)
		if err != nil {
			return helper.LogErrorAndReturn(err)
		}
		log.Printf("[INFO] Route %s/%s has been created", route.Namespace, route.Name)
	} else if err != nil {
		return helper.LogErrorAndReturn(err)
	}

	return nil
}

// GetRoute returns Route object from Openshift
func (service OpenshiftService) GetRoute(namespace string, name string) (*routeV1Api.Route, string, error) {
	route, err := service.routeClient.Routes(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Route %v in namespace %v not found", name, namespace)
		return nil, "", nil
	} else if err != nil {
		return nil, "", helper.LogErrorAndReturn(err)
	}

	var routeScheme = "http"
	if route.Spec.TLS.Termination != "" {
		routeScheme = "https"
	}
	return route, routeScheme, nil
}

// GetDeploymentConfig returns DeploymentConfig object from Openshift
func (service OpenshiftService) GetDeploymentConfig(instance v1alpha1.Nexus) (*appsV1Api.DeploymentConfig, error) {
	deploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}

	return deploymentConfig, nil
}
