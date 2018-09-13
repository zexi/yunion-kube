package namespace

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var NamespaceManager *SNamespaceManager

type SNamespaceManager struct {
	*resources.SClusterResourceManager
}

func init() {
	NamespaceManager = &SNamespaceManager{
		SClusterResourceManager: resources.NewClusterResourceManager("namespace", "namespaces"),
	}
}
