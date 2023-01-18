package nexus

import (
	"errors"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrInResponse = errors.New("received an error in http response")
)

func IsErrNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsErrInResponse(err error) bool {
	return errors.Is(err, ErrInResponse)
}
