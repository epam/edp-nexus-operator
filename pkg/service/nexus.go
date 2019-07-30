package service

import (
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
	"nexus-operator/pkg/apis/edp/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NexusProperties = "/usr/local/configs/nexus-default.properties"
	NexusPort       = 8081
)

type Client struct {
	client resty.Client
}

type NexusService interface {
	Install(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
}

func NewNexusService(platformService PlatformService, k8sClient client.Client) NexusService {
	return NexusServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

type NexusServiceImpl struct {
	platformService PlatformService
	k8sClient       client.Client
}

func (s NexusServiceImpl) Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	return &instance, nil
}

func (s NexusServiceImpl) ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	return &instance, nil
}

func (s NexusServiceImpl) Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	return &instance, nil
}

func (s NexusServiceImpl) Install(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	err := s.platformService.CreateVolume(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Volume")
	}

	_, err = s.platformService.CreateServiceAccount(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Service Account")
	}

	err = s.platformService.CreateService(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Service")
	}

	err = s.platformService.CreateConfigMapFromFile(instance, "nexus-properties", NexusProperties)
	if err != nil {
		return &instance, errors.Wrap(err, "[ERROR] Failed to create Service")
	}
	return &instance, nil
}
