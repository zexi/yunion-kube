package namespace

import (
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
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

func (m *SNamespaceManager) AllowListItems(req *common.Request) bool {
	return m.SClusterResourceManager.AllowListItems(req) || req.GetCluster().IsSharable(req.UserCred)
}

func (m *SNamespaceManager) AllowGetItem(req *common.Request, id string) bool {
	return m.AllowListItems(req)
}
