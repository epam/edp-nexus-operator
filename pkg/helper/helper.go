package helper

import (
	"context"
	"fmt"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-nexus-operator/api/common"
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

// GetValueFromSourceRef retries value from ConfigMap or Secret by SourceRef.
func GetValueFromSourceRef(
	ctx context.Context,
	sourceRef *common.SourceRef,
	namespace string,
	k8sClient client.Client,
) (string, error) {
	if sourceRef == nil {
		return "", nil
	}

	if sourceRef.ConfigMapKeyRef != nil {
		configMap := &corev1.ConfigMap{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      sourceRef.ConfigMapKeyRef.Name,
		}, configMap); err != nil {
			return "", fmt.Errorf("unable to get configmap: %w", err)
		}

		return configMap.Data[sourceRef.ConfigMapKeyRef.Key], nil
	}

	if sourceRef.SecretKeyRef != nil {
		secret := &corev1.Secret{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      sourceRef.SecretKeyRef.Name,
		}, secret); err != nil {
			return "", fmt.Errorf("unable to get secret: %w", err)
		}

		return string(secret.Data[sourceRef.SecretKeyRef.Key]), nil
	}

	return "", nil
}
