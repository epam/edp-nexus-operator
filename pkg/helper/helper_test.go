package helper

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
)

func TestLogErrorAndReturn(t *testing.T) {
	err := errors.New("test")
	assert.Equal(t, err, LogErrorAndReturn(err))
}

func TestGenerateAnnotationKey(t *testing.T) {
	str := "test"
	assert.Equal(t, fmt.Sprintf("%v/%v", spec.EdpAnnotationsPrefix, str), GenerateAnnotationKey(str))
}
