package nexus

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

type HTTPError struct {
	code    int
	message string
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("status: %d, body: %s", e.code, e.message)
}

func IsHTTPErrorCode(err error, code int) bool {
	httpError, ok := errors.Cause(err).(HTTPError)
	if !ok {
		return false
	}

	return httpError.code == code
}

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
