package openshift

import (
	"fmt"
	"github.com/epmd-edp/nexus-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/nexus-operator/v2/pkg/helper"
	nexusDefaultSpec "github.com/epmd-edp/nexus-operator/v2/pkg/service/nexus/spec"
	platformHelper "github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/helper"
	"github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/kubernetes"
	appsV1Api "github.com/openshift/api/apps/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// OpenshiftService struct for Openshift platform service
type OpenshiftService struct {
	kubernetes.K8SService

	appClient   appsV1client.AppsV1Client
	routeClient routeV1Client.RouteV1Client
}

// Init initializes OpenshiftService
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {
	err := service.K8SService.Init(config, scheme, k8sClient)
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

func (service OpenshiftService) AddKeycloakProxyToDeployConf(instance v1alpha1.Nexus, keycloakClientConf map[string][]byte) error {
	var args []string
	nexusRoute, routeScheme, err := service.GetRoute(instance.Namespace, instance.Name)
	if err != nil {
		return err
	}

	redirectUrl := fmt.Sprintf("--redirection-url=%v://%v", routeScheme, nexusRoute.Spec.Host)
	clientId := fmt.Sprintf("--client-id=%v", string(keycloakClientConf["client_id"]))
	clientSecret := fmt.Sprintf("--client-secret=%v", string(keycloakClientConf["client_secret"]))
	discoveryUrl := fmt.Sprintf("--discovery-url=%v", instance.Spec.KeycloakSpec.Url)
	upstreamUrl := fmt.Sprintf("--upstream-url=http://127.0.0.1:%v", nexusDefaultSpec.NexusPort)

	args = append(
		args,
		"--skip-openid-provider-tls-verify=true",
		discoveryUrl,
		clientId,
		clientSecret,
		"--listen=0.0.0.0:3000",
		redirectUrl,
		upstreamUrl,
		"--resources=uri=/*|roles=developer,administrator|require-any-role=true")

	containerSpec := coreV1Api.Container{
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

	oldNexusDeploymentConfig, err := service.GetDeploymentConfig(instance)
	if err != nil {
		return err
	}

	if platformHelper.ContainerInDeployConf(oldNexusDeploymentConfig.Spec.Template.Spec.Containers, containerSpec) {
		log.V(1).Info("Keycloak proxy already added!")
		return nil
	}
	oldNexusDeploymentConfig.Spec.Template.Spec.Containers = append(oldNexusDeploymentConfig.Spec.Template.Spec.Containers, containerSpec)

	_, err = service.appClient.DeploymentConfigs(instance.Namespace).Update(oldNexusDeploymentConfig)
	if err != nil {
		return err
	}

	log.Info("Keycloak proxy added.")
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
				Type: appsV1Api.DeploymentStrategyTypeRecreate,
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
									LocalObjectReference: coreV1Api.LocalObjectReference{Name: fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultPropertiesConfigMapPrefix)},
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

		log.Info(fmt.Sprintf("DeploymentConfig %v/%v has been created", deploymentConfig.Namespace, deploymentConfig.Name))
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
		log.Info(fmt.Sprintf("Route %s/%s has been created", route.Namespace, route.Name))
	} else if err != nil {
		return helper.LogErrorAndReturn(err)
	}

	return nil
}

// GetRoute returns Route object from Openshift
func (service OpenshiftService) GetRoute(namespace string, name string) (*routeV1Api.Route, string, error) {
	route, err := service.routeClient.Routes(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Route %v in namespace %v not found", name, namespace))
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

// GetRouteByCr return Route object with instance as a reference owner
func (service OpenshiftService) GetRouteByCr(instance v1alpha1.Nexus) (*routeV1Api.Route, error) {
	routeList, err := service.routeClient.Routes(instance.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't retrieve services list from the cluster")
	}
	for _, route := range routeList.Items {
		for _, owner := range route.OwnerReferences {
			if owner.UID == instance.UID {
				return &route, nil
			}
		}
	}
	return nil, nil
}

// UpdateRouteTarget performs updating route target port
func (service OpenshiftService) UpdateRouteTarget(instance v1alpha1.Nexus, targetPort intstr.IntOrString) error {
	instanceRoute, err := service.GetRouteByCr(instance)
	if err != nil || instanceRoute == nil {
		return errors.Wrapf(err, "Couldn't get route for instance %v", instance.Name)
	}
	if instanceRoute.Spec.Port != nil && instanceRoute.Spec.Port.TargetPort == targetPort {
		log.V(1).Info("Target Port %v for route route %v is already set", targetPort.StrVal, instanceRoute.Name)
		return nil
	}
	instanceRoute.Spec.Port = &routeV1Api.RoutePort{TargetPort: targetPort}

	if _, err = service.routeClient.Routes(instance.Namespace).Update(instanceRoute); err != nil {
		return err
	}
	return nil
}
