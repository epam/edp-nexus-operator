package openshift

import (
	"fmt"
	"strings"

	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1alpha1"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
	platformHelper "github.com/epam/edp-nexus-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform/kubernetes"
	routeV1Api "github.com/openshift/api/route/v1"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// OpenshiftService struct for Openshift platform service
type OpenshiftService struct {
	kubernetes.K8SService

	appClient      appsV1client.AppsV1Client
	routeClient    routeV1Client.RouteV1Client
	securityClient securityV1Client.SecurityV1Client
}

// Init initializes OpenshiftService
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {
	err := service.K8SService.Init(config, scheme, k8sClient)
	if err != nil {
		return errors.Wrap(err, "failed to initialize Kubernetes service!")
	}

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize Apps V1 Client")
	}
	service.appClient = *appClient

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize Route V1 Client")
	}
	service.routeClient = *routeClient

	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	service.securityClient = *securityClient

	return nil
}

func (service OpenshiftService) AddKeycloakProxyToDeployConf(instance v1alpha1.Nexus, args []string) error {

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

	oldNexusDeploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if platformHelper.ContainerInDeployConf(oldNexusDeploymentConfig.Spec.Template.Spec.Containers, containerSpec) {
		log.V(1).Info("Keycloak proxy is present", "Namespace", instance.Namespace, "Name", instance.Name)
		return nil
	}
	oldNexusDeploymentConfig.Spec.Template.Spec.Containers = append(oldNexusDeploymentConfig.Spec.Template.Spec.Containers, containerSpec)

	_, err = service.appClient.DeploymentConfigs(instance.Namespace).Update(oldNexusDeploymentConfig)
	if err != nil {
		return err
	}

	log.Info("Keycloak proxy added.", "Namespace", instance.Namespace, "Name", instance.Name)
	return nil

}

// GetExternalUrl returns Web URL for object and scheme from Openshift Route
func (service OpenshiftService) GetExternalUrl(namespace string, name string) (webURL, host string, scheme string, err error) {
	route, err := service.routeClient.Routes(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Route not found", "Namespace", namespace, "Name", name, "RouteName", name)
			return "", "", "", nil
		}
		return "", "", "", err
	}

	routeScheme := "http"
	if route.Spec.TLS.Termination != "" {
		routeScheme = "https"
	}
	p := strings.TrimRight(route.Spec.Path, platformHelper.UrlCutset)

	return fmt.Sprintf("%s://%s%s", routeScheme, route.Spec.Host, p), route.Spec.Host, routeScheme, nil
}

// IsDeploymentReady verifies that DeploymentConfig is ready in Openshift
func (service OpenshiftService) IsDeploymentReady(instance v1alpha1.Nexus) (res *bool, err error) {
	deploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return
	}

	t := deploymentConfig.Status.UpdatedReplicas == 1 && deploymentConfig.Status.AvailableReplicas == 1
	res = &t
	return
}

// GetRouteByCr return Route object with instance as a reference owner
func (service OpenshiftService) GetRouteByCr(instance v1alpha1.Nexus) (*routeV1Api.Route, error) {
	rl, err := service.routeClient.Routes(instance.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve services list from the cluster")
	}
	for _, r := range rl.Items {
		if r.Name == instance.Name {
			return &r, nil
		}
	}
	return nil, nil
}

// UpdateExternalTargetPath performs updating route target port
func (service OpenshiftService) UpdateExternalTargetPath(instance v1alpha1.Nexus, targetPort intstr.IntOrString) error {
	instanceRoute, err := service.GetRouteByCr(instance)
	if err != nil || instanceRoute == nil {
		return errors.Wrap(err, "couldn't get route")
	}
	if instanceRoute.Spec.Port != nil && instanceRoute.Spec.Port.TargetPort == targetPort {
		log.V(1).Info("Target Port is already set", "Namespace", instance.Namespace, "Name", instance.Name, "TargetPort", targetPort.StrVal, "Route", instanceRoute.Name)
		return nil
	}
	instanceRoute.Spec.Port = &routeV1Api.RoutePort{TargetPort: targetPort}

	_, err = service.routeClient.Routes(instance.Namespace).Update(instanceRoute)
	return err
}
