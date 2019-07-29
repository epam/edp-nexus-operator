package service

import (
	"gopkg.in/resty.v1"
	"nexus-operator/pkg/apis/edp/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	client resty.Client
}

type NexusService interface {
	Install(instance *v1alpha1.Nexus) error
	Configure(instance *v1alpha1.Nexus) error
	ExposeConfiguration(instance *v1alpha1.Nexus) error
	Integration(instance *v1alpha1.Nexus) error
}

func NewNexusService(platformService PlatformService, k8sClient client.Client) NexusService {
	return NexusServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

type NexusServiceImpl struct {
	platformService PlatformService
	k8sClient       client.Client
}

func (s NexusServiceImpl) Integration(instance *v1alpha1.Nexus) error {
	return nil
}

func (s NexusServiceImpl) ExposeConfiguration(instance *v1alpha1.Nexus) error {
	return nil
}

func (s NexusServiceImpl) Configure(instance *v1alpha1.Nexus) error {
	return nil
}

func (s NexusServiceImpl) Install(instance *v1alpha1.Nexus) error {
	return nil
}
