package limitrange

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var LimitRangeManager *SLimitRanageManager

type SLimitRanageManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	LimitRangeManager = &SLimitRanageManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("limitrange", "limitranges"),
	}
}

//func (m *SLimitRanageManager)
