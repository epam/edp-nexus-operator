package platform

import (
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/nexus-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/nexus-operator/v2/pkg/helper"
	"github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/openshift"
	routeV1Api "github.com/openshift/api/route/v1"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PlatformService interface
type PlatformService interface {
	AddKeycloakProxyToDeployConf(instance v1alpha1.Nexus, keycloakClientConf []string) error
	GetExternalUrl(namespace string, name string) (webURL string, scheme string, err error)
	UpdateRouteTarget(instance v1alpha1.Nexus, targetPort intstr.IntOrString) error
	GetRouteByCr(instance v1alpha1.Nexus) (*routeV1Api.Route, error)
	GetConfigMapData(namespace string, name string) (map[string]string, error)
	IsDeploymentReady(instance v1alpha1.Nexus) (*bool, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateSecret(instance v1alpha1.Nexus, name string, data map[string][]byte) error
	CreateService(instance v1alpha1.Nexus) error
	GetServiceByCr(instance v1alpha1.Nexus) (*coreV1Api.Service, error)
	AddPortToService(instance v1alpha1.Nexus, newPortSpec coreV1Api.ServicePort) error
	CreateVolume(instance v1alpha1.Nexus) error
	CreateServiceAccount(instance v1alpha1.Nexus) error
	CreateConfigMapFromFile(instance v1alpha1.Nexus, configMapName string, filePath string) error
	CreateConfigMapsFromDirectory(instance v1alpha1.Nexus, directoryPath string, createDedicatedConfigMaps bool) error
	CreateDeployConf(instance v1alpha1.Nexus) error
	CreateExternalEndpoint(instance v1alpha1.Nexus) error
	GetSecret(namespace string, name string) (*coreV1Api.Secret, error)
	UpdateSecret(secret *coreV1Api.Secret) error
	CreateJenkinsServiceAccount(namespace string, secretName string) error
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
	GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient,error)
}

// NewPlatformService returns platform service interface implementation
func NewPlatformService(scheme *runtime.Scheme, k8sClient *client.Client) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}

	platform := openshift.OpenshiftService{}

	err = platform.Init(restConfig, scheme, k8sClient)
	if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}
	return platform, nil
}
