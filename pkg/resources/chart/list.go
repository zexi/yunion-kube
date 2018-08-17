package chart

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/helm/data/cache"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type Chart struct {
	*cache.ChartResult
}

func (c Chart) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(c.ChartResult)
}

func ToChart(ret *cache.ChartResult) Chart {
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
	l.Charts = append(l.Charts, ToChart(obj.(*cache.ChartResult)))
}

func (man *SChartManager) List(query *cache.ChartQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	list, err := cache.ChartsList(query)
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
