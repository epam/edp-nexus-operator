package helper

const (
	UrlCutset = "!\"#$%&'()*+,-./@:;<=>[\\]^_`{|}~"
)

// GenerateLabels returns map with labels for k8s objects.
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app":                          name,
		"app.edp.epam.com/secret-type": "nexus",
	}
}
