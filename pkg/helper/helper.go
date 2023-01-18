package helper

import (
	"fmt"
	"log"

	"github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
)

func LogErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

func LogIfError(err error) {
	if err == nil {
		return
	}

	log.Printf("[ERROR] %v", err)
}

func GenerateAnnotationKey(entitySuffix string) string {
	key := fmt.Sprintf("%v/%v", spec.EdpAnnotationsPrefix, entitySuffix)
	return key
}
