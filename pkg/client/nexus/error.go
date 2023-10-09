package nexus

import "strings"

// IsErrNotFound checks if error is not found error.
func IsErrNotFound(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "not found")
}
