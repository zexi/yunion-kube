package data

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/types/helm"
)

// ICharts is an interface for managing chart data sourced from a repository index
type ICharts interface {
	// ChartFromRepo retrieves the latest version of a particular chart from a repo
	ChartFromRepo(repo, name string) (*helm.ChartPackage, error)
	// ChartVersionFromRepo retrieves a specific chart version from a repo
	ChartVersionFromRepo(repo, name, version string) (*helm.ChartPackage, error)
	// ChartVersionsFromRepo retrieves all chart versions from a repo
	ChartVersionsFromRepo(repo, name string) ([]*helm.ChartPackage, error)
	// AllFromRepo retrieves all charts from a repo
	AllFromRepo(repo string) ([]*helm.ChartPackage, error)
	// All retrieves all charts from all repos
	All() ([]*helm.ChartPackage, error)
	// Search operates against all charts/repos
	Search(params jsonutils.JSONObject) ([]*helm.ChartPackage, error)
	// Refresh charts data
	Refresh() error
	RefreshChart(repo string, chartName string) error
	DeleteChart(repo string, chartName string, version string) error
}