package chart

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var ChartManager *SChartManager

type SChartManager struct {
	*resources.SResourceBaseManager
}

func init() {
	ChartManager = &SChartManager{
		SResourceBaseManager: resources.NewResourceBaseManager("chart", "charts"),
	}
}
