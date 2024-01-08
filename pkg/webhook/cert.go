package webhook

import (
	"context"
	"fmt"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certresources "knative.dev/pkg/webhook/certificates/resources"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// secretCertsName is name of secret where ca.crt, tls.crt, tls.key will be stored after generation.
	// nolint:gosec // not sensitive data
	secretCertsName = "edp-nexus-operator-webhook-certs"
	// secretTLSKey is the name of the key associated with the secret's private key.
	secretTLSKey = "tls.key"
	// secretCACert is the name of the key associated with the certificate of the CA for the keypair.
	secretCACert = "ca.crt"
	// secretTLSCert is the name of the key associated with the secret's certificate.
	secretTLSCert = "tls.crt"
	century       = 100 * 365 * 24 * time.Hour
	// serviceName is the name of the service used to serve the webhook.
	serviceName = "edp-nexus-operator-webhook-service"
	// validatingWebHookName is the name of the ValidatingWebhookConfiguration resource used for webhook configuration.
	validatingWebHookName = "edp-nexus-operator-validating-webhook-configuration"
)

// CertData is a struct that contains certificates data.
type CertData struct {
	ServerKey  []byte
	ServerCert []byte
	CaCert     []byte
}

// NewCertData creates a new CertData struct.
func NewCertData(serverKey, serverCert, caCert []byte) *CertData {
	return &CertData{ServerKey: serverKey, ServerCert: serverCert, CaCert: caCert}
}

// CertService is a service that provides certificates for webhook.
type CertService struct {
	clientReader ctrlClient.Reader
	clientWriter ctrlClient.Writer
}

// NewCertService creates a new CertService service.
func NewCertService(clientReader ctrlClient.Reader, clientWriter ctrlClient.Writer) *CertService {
	return &CertService{
		clientReader: clientReader,
		clientWriter: clientWriter,
	}
}

// PopulateCertificates populates certificates for webhook.
func (s *CertService) PopulateCertificates(ctx context.Context, namespace string) error {
	cert, err := s.createCertsSecret(ctx, namespace, serviceName)
	if err != nil {
		return fmt.Errorf("failed to create certificates: %w", err)
	}

	return s.updateWebHookCABundle(ctx, getValidationWebHookName(namespace), cert.CaCert)
}

// createCertsSecret creates and returns a CertData with CA certificate, server certificate and key.
// serverKey and serverCert are used by the server to establish trust for clients, CA certificate is used by the
// client to verify the server authentication chain. Certificates are based on kubernetes service spec and namespace.
// After generation all certificates are stored in secret: secretCertsName.
func (s *CertService) createCertsSecret(
	ctx context.Context,
	namespace,
	serviceName string,
) (*CertData, error) {
	serKey, serCert, caCert, err := certresources.CreateCerts(
		ctx,
		serviceName,
		namespace,
		time.Now().Add(century),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create certs: %w", err)
	}

	certData := NewCertData(serKey, serCert, caCert)

	secret := &corev1.Secret{}

	err = s.clientReader.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: secretCertsName}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			secret.ObjectMeta = metav1.ObjectMeta{
				Namespace: namespace,
				Name:      secretCertsName,
			}
			secret.Data = map[string][]byte{
				secretTLSKey:  serKey,
				secretTLSCert: serCert,
				secretCACert:  caCert,
			}
			secret.Type = corev1.SecretTypeOpaque

			if err = s.clientWriter.Create(ctx, secret); err != nil {
				return nil, fmt.Errorf("failed to create secret: %w", err)
			}

			return certData, nil
		}

		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	secret.Data = map[string][]byte{
		secretTLSKey:  serKey,
		secretTLSCert: serCert,
		secretCACert:  caCert,
	}
	if err = s.clientWriter.Update(ctx, secret); err != nil {
		return nil, fmt.Errorf("failed to update secret: %w", err)
	}

	return certData, nil
}

// updateWebHookCABundle updates ValidatingWebhookConfiguration CaBundle spec with CA certificate.
func (s *CertService) updateWebHookCABundle(
	ctx context.Context,
	webHookName string,
	caBundle []byte,
) error {
	webHook := &admissionregistrationv1.ValidatingWebhookConfiguration{}

	err := s.clientReader.Get(ctx, ctrlClient.ObjectKey{Name: webHookName}, webHook)
	if err != nil {
		return fmt.Errorf("failed to get validation webHook: %w", err)
	}

	if len(webHook.Webhooks) == 0 {
		return nil
	}

	for i := range webHook.Webhooks {
		webHook.Webhooks[i].ClientConfig.CABundle = caBundle
	}

	if err = s.clientWriter.Update(ctx, webHook); err != nil {
		return fmt.Errorf("failed to update validation webHook caBundle: %w", err)
	}

	return nil
}

// getValidationWebHookName returns name of ValidatingWebhookConfiguration resource.
func getValidationWebHookName(namespace string) string {
	return fmt.Sprintf("%s-%s", validatingWebHookName, namespace)
}
