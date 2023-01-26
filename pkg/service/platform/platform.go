package platform

import (
	"fmt"
	"strings"

	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakV1Api "github.com/epam/edp-keycloak-operator/api/v1"
	edpV1 "github.com/epam/edp-nexus-operator/v2/api/v1"
	"github.com/epam/edp-nexus-operator/v2/pkg/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform/kubernetes"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform/openshift"
)

const (
	Openshift  = "openshift"
	Kubernetes = "kubernetes"
)

/*

 */
// PlatformService interface.
type PlatformService interface {
	AddKeycloakProxyToDeployConf(instance *edpV1.Nexus, args []string) error
	GetExternalUrl(namespace string, name string) (webURL string, host string, scheme string, err error)
	UpdateExternalTargetPath(instance *edpV1.Nexus, targetPort intstr.IntOrString) error
	GetConfigMapData(namespace string, name string) (map[string]string, error)
	IsDeploymentReady(instance *edpV1.Nexus) (*bool, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateSecret(instance *edpV1.Nexus, name string, data map[string][]byte) error
	GetServiceByCr(name, namespace string) (*coreV1Api.Service, error)
	AddPortToService(instance *edpV1.Nexus, newPortSpec *coreV1Api.ServicePort) error
	CreateConfigMapFromFile(instance *edpV1.Nexus, configMapName string, filePath string) error
	GetSecret(namespace string, name string) (*coreV1Api.Secret, error)
	UpdateSecret(secret *coreV1Api.Secret) error
	CreateJenkinsServiceAccount(namespace string, secretName string) error
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
	GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error)
	CreateEDPComponentIfNotExist(instance *edpV1.Nexus, url string, icon string) error
}

// NewPlatformService returns platform service interface implementation.
func NewPlatformService(platformType string, scheme *runtime.Scheme, k8sClient client.Client) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest client config: %w", helper.LogErrorAndReturn(err))
	}

	platform := openshift.OpenshiftService{}

	if err = platform.Init(restConfig, scheme, k8sClient); err != nil {
		return nil, fmt.Errorf("failed to init platform: %w", helper.LogErrorAndReturn(err))
	}

	switch strings.ToLower(platformType) {
	case Kubernetes:
		platformService := &kubernetes.K8SService{}
		if err = platformService.Init(restConfig, scheme, k8sClient); err != nil {
			return nil, fmt.Errorf("failed to initialize Kubernetes platform service: %w", err)
		}

		return platformService, nil
	case Openshift:
		platformService := &openshift.OpenshiftService{}
		if err = platformService.Init(restConfig, scheme, k8sClient); err != nil {
			return nil, fmt.Errorf("failed to initialize OpenShift platform service: %w", err)
		}

		return platformService, nil
	default:
		return nil, fmt.Errorf("platform %s is not supported: %w", platformType, err)
	}
}
