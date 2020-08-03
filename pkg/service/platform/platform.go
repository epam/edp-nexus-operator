package platform

import (
	"fmt"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/nexus-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/nexus-operator/v2/pkg/helper"
	"github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/kubernetes"
	"github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/openshift"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	Openshift  = "openshift"
	Kubernetes = "kubernetes"
)

// PlatformService interface
type PlatformService interface {
	AddKeycloakProxyToDeployConf(instance v1alpha1.Nexus, args []string) error
	GetExternalUrl(namespace string, name string) (webURL string, host string, scheme string, err error)
	UpdateExternalTargetPath(instance v1alpha1.Nexus, targetPort intstr.IntOrString) error
	GetConfigMapData(namespace string, name string) (map[string]string, error)
	IsDeploymentReady(instance v1alpha1.Nexus) (*bool, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateSecret(instance v1alpha1.Nexus, name string, data map[string][]byte) error
	CreateService(instance v1alpha1.Nexus) error
	GetServiceByCr(name, namespace string) (*coreV1Api.Service, error)
	AddPortToService(instance v1alpha1.Nexus, newPortSpec coreV1Api.ServicePort) error
	CreateVolume(instance v1alpha1.Nexus) error
	CreateServiceAccount(instance v1alpha1.Nexus) error
	CreateConfigMapFromFile(instance v1alpha1.Nexus, configMapName string, filePath string) error
	CreateConfigMapsFromDirectory(instance v1alpha1.Nexus, directoryPath string, createDedicatedConfigMaps bool) error
	CreateDeployment(instance v1alpha1.Nexus) error
	CreateExternalEndpoint(instance v1alpha1.Nexus) error
	CreateSecurityContext(ac v1alpha1.Nexus, priority int32) error
	GetSecret(namespace string, name string) (*coreV1Api.Secret, error)
	UpdateSecret(secret *coreV1Api.Secret) error
	CreateJenkinsServiceAccount(namespace string, secretName string) error
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
	GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error)
	CreateEDPComponentIfNotExist(instance v1alpha1.Nexus, url string, icon string) error
}

// NewPlatformService returns platform service interface implementation
func NewPlatformService(platformType string, scheme *runtime.Scheme, k8sClient *client.Client) (PlatformService, error) {
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
