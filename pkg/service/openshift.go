package service

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

type OpenshiftService struct {
	K8SService
}

func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	err := service.K8SService.Init(config, scheme)
	if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}
