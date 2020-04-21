package helper

import (
	coreV1Api "k8s.io/api/core/v1"
	"reflect"
)

const (
	UrlCutset = "!\"#$%&'()*+,-./@:;<=>[\\]^_`{|}~"
)

// GenerateLabels returns map with labels for k8s objects
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func ContainerInDeployConf(containers []coreV1Api.Container, newContainer coreV1Api.Container) bool {
	for _, container := range containers {
		if reflect.DeepEqual(container, newContainer) {
			return true
		}
	}
	return false
}

func PortInService(ports []coreV1Api.ServicePort, newPort coreV1Api.ServicePort) bool {
	for _, port := range ports {
		if reflect.DeepEqual(port, newPort) {
			return true
		}
	}
	return false
}
