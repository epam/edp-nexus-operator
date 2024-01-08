package helper

import (
	"fmt"
	"os"
	"strconv"
)

const (
	WatchNamespaceEnvVar   = "WATCH_NAMESPACE"
	debugModeEnvVar        = "DEBUG_MODE"
	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// GetWatchNamespace returns the namespace the operator should be watching for changes.
// If the value is not set, it returns an empty string and the operator will watch for changes in all namespaces.
func GetWatchNamespace() string {
	return os.Getenv(WatchNamespaceEnvVar)
}

// GetDebugMode returns the debug mode value.
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s value: %w", debugModeEnvVar, err)
	}

	return b, nil
}

// RunningInCluster Check whether the operator is running in cluster or locally.
func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	return !os.IsNotExist(err)
}
