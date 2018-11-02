package chart

import (
	"yunion.io/x/log"

	helmdata "yunion.io/x/yunion-kube/pkg/helm/data"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	helmtypes "yunion.io/x/yunion-kube/pkg/types/helm"
)

type Chart struct {
	*helmdata.ChartResult
}

func ToChart(ret *helmdata.ChartResult) Chart {
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
	l.Charts = append(l.Charts, ToChart(obj.(*helmdata.ChartResult)))
}

func (man *SChartManager) List(query *helmtypes.ChartQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	list, err := helmdata.ChartsList(query)
	if err != nil {
		return nil, err
	}
	log.Debugf("Get list: %#v", list)
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
