package nexus

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPError_Error(t *testing.T) {
	message := "test"
	statusCode := http.StatusNotFound
	httpError := HTTPError{
		code:    statusCode,
		message: message,
	}
	str := fmt.Sprintf("status: %d, body: %s", statusCode, message)
	assert.Equal(t, str, httpError.Error())
}
