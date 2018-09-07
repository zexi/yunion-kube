package statefulset

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var StatefulSetManager *SStatefuleSetManager

type SStatefuleSetManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	StatefulSetManager = &SStatefuleSetManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("statefulset", "statefulsets"),
	}
}
