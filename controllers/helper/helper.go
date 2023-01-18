package helper

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	watchNamespaceEnvVar         = "WATCH_NAMESPACE"
	debugModeEnvVar              = "DEBUG_MODE"
	inClusterNamespacePath       = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	platformType                 = "PLATFORM_TYPE"
	FailureReconciliationTimeout = time.Second * 10
)

func GetPlatformTypeEnv() string {
	return os.Getenv(platformType)
}

// GetWatchNamespace returns the namespace the operator should be watching for changes.
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}

	return ns, nil
}

// GetDebugMode returns the debug mode value.
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, fmt.Errorf("failed to parse debug mode value: %w", err)
	}

	return b, nil
}

// RunningInCluster checks whether the operator is running in cluster or locally.
func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)

	return !os.IsNotExist(err)
}

func ContainsString(slice []string, s string) bool {
	for i := range slice {
		if slice[i] == s {
			return true
		}
	}

	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for i := range slice {
		if slice[i] == s {
			continue
		}

		result = append(result, slice[i])
	}

	return
}
