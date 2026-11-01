package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/secretref"
)

const redactedValue = "<redacted>"

// resolvedAuth is the wire form of httpClient.authentication.
type resolvedAuth struct {
	wire *repository.HTTPClientAuthenticationWithPreemptive
	// secretPassword is the password read from the Secret; empty for a plain password.
	secretPassword string
}

// resolveAuthentication validates the authentication view and reads passwordRef.
// The rules are the same as the CEL rules of the CRD.
func resolveAuthentication(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	auth *nexus.ProxyAuth,
	maven bool,
) (*resolvedAuth, error) {
	if err := validateAuthentication(auth, maven); err != nil {
		return nil, err
	}

	password := auth.Password
	secretPassword := ""

	if auth.PasswordRef != nil {
		value, err := secretref.GetValueFromSecretKeySelector(ctx, auth.PasswordRef, namespace, k8sClient)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve passwordRef: %w", err)
		}

		if err = validateSecretValue(auth.PasswordRef.Name, auth.PasswordRef.Key, value); err != nil {
			return nil, err
		}

		password = value
		secretPassword = value
	}

	return &resolvedAuth{
		secretPassword: secretPassword,
		wire: &repository.HTTPClientAuthenticationWithPreemptive{
			NTLMDomain: auth.NTLMDomain,
			NTLMHost:   auth.NTLMHost,
			Password:   password,
			Type:       repository.HTTPClientAuthenticationType(auth.Type),
			Username:   auth.Username,
			Preemptive: auth.Preemptive,
		},
	}, nil
}

func validateAuthentication(auth *nexus.ProxyAuth, maven bool) error {
	if auth.PasswordRef != nil && (auth.PasswordRef.Name == "" || auth.PasswordRef.Key == "") {
		return errors.New("invalid httpClient.authentication: passwordRef name and key must not be empty")
	}

	hasPassword := auth.Password != ""
	hasRef := auth.PasswordRef != nil

	if hasPassword && hasRef {
		return errors.New("invalid httpClient.authentication: password and passwordRef are mutually exclusive")
	}

	if maven {
		if (auth.Username != "") != (hasPassword || hasRef) {
			return errors.New("invalid httpClient.authentication: " +
				"username requires password or passwordRef, and a password requires username")
		}

		return nil
	}

	if !hasPassword && !hasRef {
		return errors.New("invalid httpClient.authentication: one of password or passwordRef must be set")
	}

	return nil
}

func validateSecretValue(name, key, value string) error {
	if value == "" {
		return fmt.Errorf("secret %s key %s is empty", name, key)
	}

	if strings.TrimSpace(value) != value {
		return fmt.Errorf("secret %s key %s has leading or trailing whitespace", name, key)
	}

	return nil
}

// redactError returns err with every occurrence of value replaced, in plain and in JSON-escaped form
// (with and without HTML escaping). An empty value leaves err as is.
func redactError(err error, value string) error {
	if err == nil || value == "" {
		return err
	}

	msg := err.Error()

	// The JSON forms go first: a plain-form replacement would split an escape sequence.
	for _, form := range jsonForms(value) {
		msg = strings.ReplaceAll(msg, form, redactedValue)
	}

	return errors.New(strings.ReplaceAll(msg, value, redactedValue))
}

// jsonForms returns value as the inside of a JSON string literal: HTML-escaped (json.Marshal) and not.
func jsonForms(value string) []string {
	forms := make([]string, 0, 2)

	if encoded, err := json.Marshal(value); err == nil {
		forms = append(forms, string(encoded[1:len(encoded)-1]))
	}

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(value); err == nil {
		encoded := strings.TrimSuffix(buf.String(), "\n")
		forms = append(forms, encoded[1:len(encoded)-1])
	}

	return forms
}
