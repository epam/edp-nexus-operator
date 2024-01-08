package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCertService_PopulateCertificates(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	defaultNs := "default"

	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, admissionregistrationv1.AddToScheme(scheme))

	tests := []struct {
		name      string
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		wantCheck func(t *testing.T, c client.Client)
	}{
		{
			name: "should create secret and update webhook CaBundle successfully",
			objects: []client.Object{
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metaV1.ObjectMeta{
						Name: getValidationWebHookName(defaultNs),
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{
						{
							Name: "validate",
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      serviceName,
						Namespace: defaultNs,
					},
				},
			},
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, c client.Client) {
				secret := &corev1.Secret{}
				err := c.Get(
					context.Background(),
					client.ObjectKey{
						Namespace: defaultNs,
						Name:      secretCertsName,
					},
					secret,
				)

				require.NoError(t, err)
				require.NotEmpty(t, secret.Data[secretTLSKey])
				require.NotEmpty(t, secret.Data[secretCACert])
				require.NotEmpty(t, secret.Data[secretTLSCert])

				webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
				err = c.Get(
					context.Background(),
					client.ObjectKey{
						Name: getValidationWebHookName(defaultNs),
					},
					webhook,
				)

				require.NoError(t, err)
				require.NotEmpty(t, webhook.Webhooks)
				require.NotEmpty(t, webhook.Webhooks[0].ClientConfig.CABundle)
			},
		},
		{
			name: "should update secret and update webhook CaBundle successfully",
			objects: []client.Object{
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metaV1.ObjectMeta{
						Name: getValidationWebHookName(defaultNs),
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{
						{
							Name: "validate",
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      serviceName,
						Namespace: defaultNs,
					},
				},
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      secretCertsName,
						Namespace: defaultNs,
					},
				},
			},
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, c client.Client) {
				secret := &corev1.Secret{}
				err := c.Get(
					context.Background(),
					client.ObjectKey{
						Namespace: defaultNs,
						Name:      secretCertsName,
					},
					secret,
				)

				require.NoError(t, err)
				require.NotEmpty(t, secret.Data[secretTLSKey])
				require.NotEmpty(t, secret.Data[secretCACert])
				require.NotEmpty(t, secret.Data[secretTLSCert])

				webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
				err = c.Get(
					context.Background(),
					client.ObjectKey{
						Name: getValidationWebHookName(defaultNs),
					},
					webhook,
				)

				require.NoError(t, err)
				require.NotEmpty(t, webhook.Webhooks)
				require.NotEmpty(t, webhook.Webhooks[0].ClientConfig.CABundle)
			},
		},
		{
			name: "empty webhook",
			objects: []client.Object{
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metaV1.ObjectMeta{
						Name: getValidationWebHookName(defaultNs),
					},
				},
				&corev1.Service{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      serviceName,
						Namespace: defaultNs,
					},
				},
			},
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, c client.Client) {
				secret := &corev1.Secret{}
				err := c.Get(
					context.Background(),
					client.ObjectKey{
						Namespace: defaultNs,
						Name:      secretCertsName,
					},
					secret,
				)

				require.NoError(t, err)
				require.NotEmpty(t, secret.Data[secretTLSKey])
				require.NotEmpty(t, secret.Data[secretCACert])
				require.NotEmpty(t, secret.Data[secretTLSCert])

				webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
				err = c.Get(
					context.Background(),
					client.ObjectKey{
						Name: getValidationWebHookName(defaultNs),
					},
					webhook,
				)

				require.NoError(t, err)
				require.Empty(t, webhook.Webhooks)
			},
		},
		{
			name: "validatingWebhookConfiguration resource not found",
			objects: []client.Object{
				&corev1.Service{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      serviceName,
						Namespace: defaultNs,
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get validation webHook")
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()

			s := NewCertService(k8sClient, k8sClient)
			err := s.PopulateCertificates(context.Background(), defaultNs)

			tt.wantErr(t, err)

			if tt.wantCheck != nil {
				tt.wantCheck(t, k8sClient)
			}
		})
	}
}
