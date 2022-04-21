package nexus

import (
	"context"

	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

func (nc *Client) requestWithContext(ctx context.Context) *resty.Request {
	return nc.resty.R().SetContext(ctx)
}

func checkRestyResponse(response *resty.Response, err error) error {
	if err != nil {
		return errors.Wrap(err, "response error")
	}

	if response == nil {
		return errors.New("empty response")
	}

	if response.IsError() {
		return HTTPError{message: response.String(), code: response.StatusCode()}
	}

	return nil
}
