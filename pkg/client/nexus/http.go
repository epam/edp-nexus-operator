package nexus

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/resty.v1"
)

func (nc *Client) requestWithContext(ctx context.Context) *resty.Request {
	return nc.resty.R().SetContext(ctx)
}

func checkRestyResponse(response *resty.Response, err error) error {
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	if response == nil {
		return errors.New("empty response")
	}

	if response.IsError() {
		return fmt.Errorf("failed to validate response: %w, response: %s, code: %d", ErrInResponse, response.String(), response.StatusCode())
	}

	return nil
}
