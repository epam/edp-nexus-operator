package helper

import (
	"fmt"
	"log"
	"nexus-operator/pkg/service/nexus/spec"
	"os"
	"path/filepath"
)

func LogErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

func GetExecutableFilePath() string {
	executableFilePath, err := os.Executable()
	if err != nil {
		LogErrorAndReturn(err)
	}
	return filepath.Dir(executableFilePath)
}

func GenerateAnnotationKey(entitySuffix string) string {
	key := fmt.Sprintf("%v/%v", spec.EdpAnnotationsPrefix, entitySuffix)
	return key
}