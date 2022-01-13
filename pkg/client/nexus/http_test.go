package nexus

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
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
