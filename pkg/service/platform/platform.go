package platform

import (
	appsV1Api "github.com/openshift/api/apps/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"nexus-operator/pkg/apis/edp/v1alpha1"
	"nexus-operator/pkg/helper"
	"nexus-operator/pkg/service/platform/openshift"
)

// PlatformService interface
type PlatformService interface {
	GetRoute(namespace string, name string) (*routeV1Api.Route, string, error)
	GetConfigMapData(namespace string, name string) (map[string]string, error)
	GetDeploymentConfig(instance v1alpha1.Nexus) (*appsV1Api.DeploymentConfig, error)
	CreateService(instance v1alpha1.Nexus) error
	CreateVolume(instance v1alpha1.Nexus) error
	CreateServiceAccount(instance v1alpha1.Nexus) (*coreV1Api.ServiceAccount, error)
	CreateConfigMapFromFile(instance v1alpha1.Nexus, configMapName string, filePath string) error
	CreateConfigMapsFromDirectory(instance v1alpha1.Nexus, directoryPath string) error
	CreateDeployConf(instance v1alpha1.Nexus) error
	CreateExternalEndpoint(instance v1alpha1.Nexus) error
}

// NewPlatformService returns platform service interface implementation
func NewPlatformService(scheme *runtime.Scheme) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}

	platform := openshift.OpenshiftService{}

	err = platform.Init(restConfig, scheme)
	if err != nil {
		return nil, helper.LogErrorAndReturn(err)
	}
	return platform, nil
}
