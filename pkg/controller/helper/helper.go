package helper

import "os"

const platformType string = "PLATFORM_TYPE"


func GetPlatformTypeEnv() string {
	return os.Getenv(platformType)
}

