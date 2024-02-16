package nexus

import (
	"errors"
	"strings"
)

// IsErrNotFound checks if error is not found error.
func IsErrNotFound(err error) bool {
	if err == nil {
		return false
	}

	if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Unable to find") {
		return true
	}

	if errors.Is(err, ErrNotFound) {
		return true
	}

	return false
}
