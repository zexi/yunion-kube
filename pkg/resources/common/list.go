package common

import (
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type BaseList struct {
	*dataselect.ListMeta
	Cluster api.ICluster
}

func NewBaseList(cluster api.ICluster) *BaseList {
	return &BaseList{
		ListMeta: dataselect.NewListMeta(),
		Cluster:  cluster,
	}
}

func (l *BaseList) GetCluster() api.ICluster {
	return l.Cluster
}
