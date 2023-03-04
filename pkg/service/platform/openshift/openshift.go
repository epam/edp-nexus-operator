package openshift

import (
	"context"
	"fmt"
	"os"
	"strings"

	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/epam/edp-nexus-operator/v2/api/v1"
	platformHelper "github.com/epam/edp-nexus-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform/kubernetes"
)

var log = ctrl.Log.WithName("platform")

type OpenshiftClient interface {
	appsV1client.AppsV1Interface
}

type RouteClient interface {
	routeV1Client.RouteV1Interface
}

type SecurityClient interface {
	securityV1Client.SecurityV1Interface
}

// OpenshiftService struct for Openshift platform service.
type OpenshiftService struct {
	kubernetes.K8SService

	appClient      OpenshiftClient
	routeClient    RouteClient
	securityClient SecurityClient
}

const (
	deploymentTypeEnvName           = "DEPLOYMENT_TYPE"
	deploymentConfigsDeploymentType = "deploymentConfigs"
	crNameKey                       = "Name"
	crNamespaceKey                  = "Namespace"
)

// Init initializes OpenshiftService.
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient client.Client) error {
	err := service.K8SService.Init(config, scheme, k8sClient)
	if err != nil {
		return fmt.Errorf("failed to initialize Kubernetes service!: %w", err)
	}

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to initialize Apps V1 Client: %w", err)
	}

	service.appClient = appClient

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to initialize Route V1 Client: %w", err)
	}

	service.routeClient = routeClient

	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create new config: %w", err)
	}

	service.securityClient = securityClient

	return nil
}

// GetExternalUrl returns Web URL for object and scheme from Openshift Route.
func (service *OpenshiftService) GetExternalUrl(namespace, name string) (url, host, schema string, err error) {
	route, err := service.routeClient.Routes(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		log.Info("Route not found", crNamespaceKey, namespace, crNameKey, name, "RouteName", name)
		return "", "", "", nil
	}

	if err != nil {
		return "", "", "", fmt.Errorf("failed to get route: %w", err)
	}

	routeScheme := "http"
	if route.Spec.TLS.Termination != "" {
		routeScheme = "https"
	}

	p := strings.TrimRight(route.Spec.Path, platformHelper.UrlCutset)

	return fmt.Sprintf("%s://%s%s", routeScheme, route.Spec.Host, p), route.Spec.Host, routeScheme, nil
}

// IsDeploymentReady verifies that DeploymentConfig is ready in Openshift.
func (service *OpenshiftService) IsDeploymentReady(instance *v1.Nexus) (*bool, error) {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		deploymentConfig, err := service.appClient.
			DeploymentConfigs(instance.Namespace).
			Get(context.TODO(), instance.Name, metav1.GetOptions{})
		if err != nil {
			return getBoolP(false), fmt.Errorf("failed to get deployment config %s: %w", instance.Name, err)
		}

		t := deploymentConfig.Status.UpdatedReplicas == 1 && deploymentConfig.Status.AvailableReplicas == 1

		return getBoolP(t), nil
	}

	ready, err := service.K8SService.IsDeploymentReady(instance)
	if err != nil {
		return getBoolP(false), fmt.Errorf("failed to check if deployment is ready: %w", err)
	}

	return ready, nil
}

func getBoolP(val bool) *bool {
	return &val
}
