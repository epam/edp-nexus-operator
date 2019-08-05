package helper

import (
	"regexp"
)

func FormateNexusScript(content string) string {
	re := regexp.MustCompile(`\r?\n`)
	formattedScript := re.ReplaceAllString(content, "\\n")

	re = regexp.MustCompile(`\"`)
	formattedScript = re.ReplaceAllString(formattedScript, "\\\"")
	return formattedScript
}
