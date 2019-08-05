package helper

import "log"

func LogErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}
