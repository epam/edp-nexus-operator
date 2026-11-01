package secretref

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-nexus-operator/api/common"
)

// GetValueFromSecretKeySelector returns the value of the Secret key named by the selector.
// A nil selector returns an empty value. Errors name the Secret and the key, never the value.
func GetValueFromSecretKeySelector(
	ctx context.Context,
	selector *common.SecretKeySelector,
	namespace string,
	k8sClient client.Client,
) (string, error) {
	if selector == nil {
		return "", nil
	}

	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      selector.Name,
	}, secret); err != nil {
		return "", fmt.Errorf("unable to get secret %s: %w", selector.Name, err)
	}

	val, ok := secret.Data[selector.Key]
	if !ok {
		return "", fmt.Errorf("secret %s does not contain key %s", selector.Name, selector.Key)
	}

	return string(val), nil
}
