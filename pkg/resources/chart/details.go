package chart

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/helm/data/cache"
)

func (man *SChartManager) Show(repoName, chartName, version string) (jsonutils.JSONObject, error) {
	detail, err := cache.ChartShowDetails(repoName, chartName, version)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(detail), nil
}
