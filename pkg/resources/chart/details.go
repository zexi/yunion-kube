package chart

import (
	"encoding/base64"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/options"
)

func (man *SChartManager) Show(repoName, chartName, version string) (interface{}, error) {
	return man.GetDetails(repoName, chartName, version)
}

func (man *SChartManager) GetDetails(repoName, chartName, version string) (*apis.ChartDetail, error) {
	chart, err := helm.NewChartClient(options.Options.HelmDataDir).Show(repoName, chartName, version)
	if err != nil {
		return nil, err
	}
	readmeStr := ""
	readmeFile := helm.FindChartReadme(chart)
	if readmeFile != nil {
		readmeStr = base64.StdEncoding.EncodeToString(readmeFile.Data)
	}
	return &apis.ChartDetail{
		Repo:   repoName,
		Name:   chart.Name(),
		Chart:  chart,
		Readme: readmeStr,
	}, nil
}
