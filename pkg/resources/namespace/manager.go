package namespace

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var NamespaceManager *SNamespaceManager

type SNamespaceManager struct {
	*resources.SResourceBaseManager
}

func init() {
	NamespaceManager = &SNamespaceManager{
		SResourceBaseManager: resources.NewResourceBaseManager("namespace", "namespaces"),
	}
}
