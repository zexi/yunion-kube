package chart

import (
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type Chart struct {
	*apis.ChartResult
}

func ToChart(ret *apis.ChartResult) Chart {
	return Chart{ret}
}

type ChartList struct {
	*dataselect.ListMeta
	Charts []Chart
}

func (l *ChartList) GetResponseData() interface{} {
	return l.Charts
}

func (l *ChartList) Append(obj interface{}) {
	l.Charts = append(l.Charts, ToChart(obj.(*apis.ChartResult)))
}

func (man *SChartManager) List(query *apis.ChartListInput, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	cli := helm.NewChartClient(options.Options.HelmDataDir)
	list, err := cli.SearchRepo(*query, "") //, query.RepoUrl, query.Keyword)
	if err != nil {
		return nil, err
	}
	chartList := &ChartList{
		ListMeta: dataselect.NewListMeta(),
		Charts:   make([]Chart, 0),
	}
	err = dataselect.ToResourceList(
		chartList,
		list,
		dataselect.NewChartDataCell,
		dsQuery,
	)
	return chartList, err
}
