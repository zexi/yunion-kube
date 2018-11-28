package statefulset

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var StatefulSetManager *SStatefuleSetManager

type SStatefuleSetManager struct {
	*resources.SNamespaceResourceManager
}

func (m *SStatefuleSetManager) IsRawResource() bool {
	return false
}

func init() {
	StatefulSetManager = &SStatefuleSetManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("statefulset", "statefulsets"),
	}
}
