package nexus

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// WithAuthentication returns the JSON object form of data with httpClient.authentication replaced by auth.
// Numbers keep their decimal form. data is not modified.
func WithAuthentication(data any, auth *repository.HTTPClientAuthenticationWithPreemptive) (map[string]any, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal repository data: %w", err)
	}

	body := map[string]any{}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	if err = dec.Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode repository data: %w", err)
	}

	if body == nil {
		return nil, errors.New("repository data is not a JSON object")
	}

	httpClient, ok := body["httpClient"].(map[string]any)
	if !ok {
		httpClient = map[string]any{}
		body["httpClient"] = httpClient
	}

	httpClient["authentication"] = auth

	return body, nil
}
