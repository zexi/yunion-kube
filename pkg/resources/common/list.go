package common

import (
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type BaseList struct {
	*dataselect.ListMeta
	Cluster apis.ICluster
}

func NewBaseList(cluster apis.ICluster) *BaseList {
	return &BaseList{
		ListMeta: dataselect.NewListMeta(),
		Cluster:  cluster,
	}
}

func (l *BaseList) GetCluster() apis.ICluster {
	return l.Cluster
}
