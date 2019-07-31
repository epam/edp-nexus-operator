package service

import (
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"nexus-operator/pkg/apis/edp/v1alpha1"
)

type PlatformService interface {
	CreateService(sonar v1alpha1.Nexus) error
	CreateVolume(sonar v1alpha1.Nexus) error
	CreateServiceAccount(sonar v1alpha1.Nexus) (*coreV1Api.ServiceAccount, error)
	CreateConfigMapFromFile(sonar v1alpha1.Nexus, configMapName string, filePath string) error
	CreateDeployConf(ac v1alpha1.Nexus) error
	CreateExternalEndpoint(ac v1alpha1.Nexus) error
}

func NewPlatformService(scheme *runtime.Scheme) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, logErrorAndReturn(err)
	}

	platform := OpenshiftService{}

	err = platform.Init(restConfig, scheme)
	if err != nil {
		return nil, logErrorAndReturn(err)
	}
	return platform, nil
}

func generateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func logErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}
