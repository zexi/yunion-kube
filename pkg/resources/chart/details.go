package chart

import (
//helmdata "yunion.io/x/yunion-kube/pkg/helm/data"
)

func (man *SChartManager) Show(repoName, chartName, version string) (interface{}, error) {
	return helmdata.ChartShowDetails(repoName, chartName, version)
}
