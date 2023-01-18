package helper

import (
	"reflect"

	coreV1Api "k8s.io/api/core/v1"
)

const (
	UrlCutset = "!\"#$%&'()*+,-./@:;<=>[\\]^_`{|}~"
)

// GenerateLabels returns map with labels for k8s objects.
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func ContainerInDeployConf(containers []coreV1Api.Container, newContainer *coreV1Api.Container) bool {
	for i := range containers {
		if containers[i].Name == newContainer.Name {
			return true
		}
	}

	return false
}

func PortInService(ports []coreV1Api.ServicePort, newPort *coreV1Api.ServicePort) bool {
	for i := range ports {
		if reflect.DeepEqual(&ports[i], newPort) {
			return true
		}
	}

	return false
}
