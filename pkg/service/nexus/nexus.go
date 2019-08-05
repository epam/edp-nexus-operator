package nexus

import (
	"fmt"
	"github.com/pkg/errors"
	"nexus-operator/pkg/apis/edp/v1alpha1"
	"nexus-operator/pkg/client/nexus"
	"nexus-operator/pkg/helper"
	nexusDefaultSpec "nexus-operator/pkg/service/nexus/spec"
	"nexus-operator/pkg/service/platform"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NexusService interface for Nexus EDP component
type NexusService interface {
	Install(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	IsDeploymentConfigReady(instance v1alpha1.Nexus) (bool, error)
}

// NewNexusService function that returns NexusService implementation
func NewNexusService(platformService platform.PlatformService, k8sClient client.Client) NexusService {
	return NexusServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

// NexusServiceImpl struct fo Nexus EDP Component
type NexusServiceImpl struct {
	platformService platform.PlatformService
	k8sClient       client.Client
	nexusClient     nexus.NexusClient
}

// IsDeploymentConfigReady check if DC for Nexus is ready
func (n NexusServiceImpl) IsDeploymentConfigReady(instance v1alpha1.Nexus) (bool, error) {
	nexusIsReady := false
	nexusDc, err := n.platformService.GetDeploymentConfig(instance)
	if err != nil {
		return nexusIsReady, helper.LogErrorAndReturn(err)
	}
	if nexusDc.Status.AvailableReplicas == 1 {
		nexusIsReady = true
	}
	return nexusIsReady, nil
}

// Integration performs integration Nexus with other EDP components
func (n NexusServiceImpl) Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	return &instance, nil
}

// ExposeConfiguration performs exposing Nexus configuration for other EDP components
func (n NexusServiceImpl) ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	return &instance, nil
}

// Configure performs self-configuration of Nexus
func (n NexusServiceImpl) Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	nexusRoute, nexusRouteScheme, err := n.platformService.GetRoute(instance.Namespace, instance.Name)
	if err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("[ERROR] Failed to get route for %v/%v", instance.Namespace, instance.Name))
	}
	nexusApiUrl := fmt.Sprintf("%v://%v/%v", nexusRouteScheme, nexusRoute.Spec.Host, nexusDefaultSpec.NexusRestApiUrlPath)
	err = n.nexusClient.InitNewRestClient(&instance, nexusApiUrl, nexusDefaultSpec.NexusDefaultAdminUser, nexusDefaultSpec.NexusDefaultAdminPassword)
	if err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("[ERROR] Failed to initialize Nexus client for %v/%v", instance.Namespace, instance.Name))
	}

	err = n.nexusClient.WaitForStatusIsUp(60, 10)
	if err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("[ERROR] Failed to check status for %v/%v", instance.Namespace, instance.Name))
	}

	err = n.nexusClient.DeclareDefaultScripts(nexusDefaultSpec.NexusScriptsPath)
	if err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("[ERROR] Failed to upload default scripts for %v/%v", instance.Namespace, instance.Name))
	}

	defaultScriptsAreDeclared, err := n.nexusClient.AreDefaultScriptsDeclared(nexusDefaultSpec.NexusScriptsPath)
	if !defaultScriptsAreDeclared || err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("[ERROR] Default scripts for %v/%v are not uploaded yet", instance.Namespace, instance.Name))
	}
	return &instance, nil
}

// Install performs installation of Nexus
func (n NexusServiceImpl) Install(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	err := n.platformService.CreateVolume(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Volume")
	}

	_, err = n.platformService.CreateServiceAccount(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Service Account")
	}

	err = n.platformService.CreateService(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Service")
	}

	err = n.platformService.CreateConfigMapFromFile(instance, "nexus-properties", nexusDefaultSpec.NexusProperties)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Config Map")
	}

	err = n.platformService.CreateDeployConf(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Deployment Config")
	}

	err = n.platformService.CreateExternalEndpoint(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create External Route")
	}

	return &instance, nil
}
