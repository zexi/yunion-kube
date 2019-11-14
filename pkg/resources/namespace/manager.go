package namespace

import (
	"k8s.io/api/core/v1"

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

func (m *SNamespaceManager) ValidateCreateData(req *common.Request) error {
	return m.SClusterResourceManager.ValidateCreateData(req)
}

func (m *SNamespaceManager) Create(req *common.Request) (interface{}, error) {
	objMeta, err := common.GetK8sObjectCreateMeta(req.Data)
	if err != nil {
		return nil, err
	}
	ns := &v1.Namespace{
		ObjectMeta: *objMeta,
	}
	return req.GetK8sClient().CoreV1().Namespaces().Create(ns)
}
