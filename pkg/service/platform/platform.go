package platform

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"

	nexusApi "github.com/epam/edp-nexus-operator/v2/api/edp/v1"
	"github.com/epam/edp-nexus-operator/v2/pkg/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform/kubernetes"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform/openshift"
)

const (
	Openshift  = "openshift"
	Kubernetes = "kubernetes"
)

// PlatformService interface.
type PlatformService interface {
	AddKeycloakProxyToDeployConf(instance nexusApi.Nexus, args []string) error
	GetExternalUrl(namespace string, name string) (webURL string, host string, scheme string, err error)
	UpdateExternalTargetPath(instance nexusApi.Nexus, targetPort intstr.IntOrString) error
	GetConfigMapData(namespace string, name string) (map[string]string, error)
	IsDeploymentReady(instance nexusApi.Nexus) (*bool, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateSecret(instance nexusApi.Nexus, name string, data map[string][]byte) error
	GetServiceByCr(name, namespace string) (*coreV1Api.Service, error)
	AddPortToService(instance nexusApi.Nexus, newPortSpec coreV1Api.ServicePort) error
	CreateConfigMapFromFile(instance nexusApi.Nexus, configMapName string, filePath string) error
	GetSecret(namespace string, name string) (*coreV1Api.Secret, error)
	UpdateSecret(secret *coreV1Api.Secret) error
	CreateJenkinsServiceAccount(namespace string, secretName string) error
	CreateKeycloakClient(kc *keycloakApi.KeycloakClient) error
	GetKeycloakClient(name string, namespace string) (keycloakApi.KeycloakClient, error)
	CreateEDPComponentIfNotExist(instance nexusApi.Nexus, url string, icon string) error
}

// NewPlatformService returns platform service interface implementation.
func NewPlatformService(platformType string, scheme *runtime.Scheme, k8sClient client.Client) (PlatformService, error) {
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

	switch strings.ToLower(platformType) {
	case Kubernetes:
		platformService := kubernetes.K8SService{}
		err = platformService.Init(restConfig, scheme, k8sClient)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to initialize Kubernetes platform service!")
		}

		return platformService, nil
	case Openshift:
		platformService := openshift.OpenshiftService{}
		err = platformService.Init(restConfig, scheme, k8sClient)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to initialize OpenShift platform service!")
		}

		return platformService, nil
	default:
		err := errors.New(fmt.Sprintf("Platform %s is not supported!", platformType))
		return nil, err
	}
}
