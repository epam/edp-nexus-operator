package helper

import (
	"log"
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
