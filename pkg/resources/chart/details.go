package chart

import (
	"yunion.io/x/yunion-kube/pkg/helm/data/cache"
)

func (man *SChartManager) Show(repoName, chartName, version string) (interface{}, error) {
	return cache.ChartShowDetails(repoName, chartName, version)
}
