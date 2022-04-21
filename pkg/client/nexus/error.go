package nexus

import (
	"fmt"

	"github.com/pkg/errors"
)

type ErrNotFound string

func (e ErrNotFound) Error() string {
	return string(e)
}

func IsErrNotFound(err error) bool {
	_, ok := errors.Cause(err).(ErrNotFound)
	return ok
}

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
