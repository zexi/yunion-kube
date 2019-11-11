package chart

import (
	"path/filepath"

	"encoding/base64"

	"helm.sh/helm/v3/pkg/chart"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/options"
)

func (man *SChartManager) Show(repoName, chartName, version string) (interface{}, error) {
	return man.GetDetails(repoName, chartName, version)
}

func (man *SChartManager) GetDetails(repoName, chartName, version string) (*apis.ChartDetail, error) {
	chObj, err := helm.NewChartClient(options.Options.HelmDataDir).Show(repoName, chartName, version)
	if err != nil {
		return nil, err
	}
	readmeStr := ""
	readmeFile := helm.FindChartReadme(chObj)
	if readmeFile != nil {
		readmeStr = base64.StdEncoding.EncodeToString(readmeFile.Data)
	}
	files := make([]*chart.File, len(chObj.Raw))
	for idx, rf := range chObj.Raw {
		files[idx] = &chart.File{
			Name: filepath.Join(chObj.Name(), rf.Name),
			Data: rf.Data,
		}
	}
	return &apis.ChartDetail{
		Repo:   repoName,
		Name:   chObj.Name(),
		Chart:  chObj,
		Readme: readmeStr,
		Files:  files,
	}, nil
}
