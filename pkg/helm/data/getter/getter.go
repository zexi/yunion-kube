package getter

import (
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
)

// All finds all of the registered getters as a list of Provider instances.
// Currently the build-in http/https getter and the discovered
// plugins with downloader notations are collected.
func All(settings environment.EnvSettings) getter.Providers {
	result := getter.Providers{
		{
			Schemes: []string{"http", "https"},
			New:     newHTTPGetter,
		},
	}
	return result
}
