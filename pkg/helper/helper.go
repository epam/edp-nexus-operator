package helper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
)

func LogErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

func GetExecutableFilePath() string {
	executableFilePath, err := os.Executable()
	if err != nil {
		_ = LogErrorAndReturn(err)
	}
	return filepath.Dir(executableFilePath)
}

func GenerateAnnotationKey(entitySuffix string) string {
	key := fmt.Sprintf("%v/%v", spec.EdpAnnotationsPrefix, entitySuffix)
	return key
}
