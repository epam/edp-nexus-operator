package nexus

import "github.com/epam/edp-nexus-operator/api/v1alpha1"

// GetHTTPAuth returns the HTTP client authentication of RepoData.Data.
// It returns nil for hosted and group repositories and for proxies without authentication.
func GetHTTPAuth(data interface{}) v1alpha1.HTTPAuthCredentials {
	provider, ok := data.(interface {
		HTTPAuth() v1alpha1.HTTPAuthCredentials
	})
	if !ok {
		return nil
	}

	return provider.HTTPAuth()
}
