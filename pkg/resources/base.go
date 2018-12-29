package resources

import (
	"fmt"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

type SResourceBaseManager struct {
	keyword       string
	keywordPlural string
}

func NewResourceBaseManager(keyword, keywordPlural string) *SResourceBaseManager {
	return &SResourceBaseManager{
		keyword:       keyword,
		keywordPlural: keywordPlural,
	}
}

func (m *SResourceBaseManager) Keyword() string {
	return m.keyword
}

func (m *SResourceBaseManager) KeywordPlural() string {
	return m.keywordPlural
}

func (m *SResourceBaseManager) InNamespace() bool {
	return false
}

func (m *SResourceBaseManager) AllowListItems(req *common.Request) bool {
	log.Errorf("AllowListItems not implemented")
	return false
}

func (m *SResourceBaseManager) List(req *common.Request) (common.ListResource, error) {
	return nil, fmt.Errorf("List not implemented")
}

func (m *SResourceBaseManager) AllowGetItem(req *common.Request, id string) bool {
	return req.UserCred.IsSystemAdmin()
}

func (m *SResourceBaseManager) Get(req *common.Request, id string) (interface{}, error) {
	return nil, fmt.Errorf("Get resource not implemented")
}

func (m *SResourceBaseManager) AllowCreateItem(req *common.Request) bool {
	return req.AllowCreateItem()
}

func (m *SResourceBaseManager) ValidateCreateData(req *common.Request) error {
	return nil
}

func (m *SResourceBaseManager) Create(req *common.Request) (interface{}, error) {
	return nil, fmt.Errorf("Create not implemented")
}

func (m *SResourceBaseManager) AllowUpdateItem(req *common.Request, id string) bool {
	return m.AllowDeleteItem(req, id)
}

func (m *SResourceBaseManager) Update(req *common.Request, id string) (interface{}, error) {
	return nil, fmt.Errorf("Update resource not implemented")
}

func (m *SResourceBaseManager) AllowDeleteItem(req *common.Request, id string) bool {
	cred := req.UserCred
	if cred.IsSystemAdmin() {
		return true
	}
	return false
}

func (m *SResourceBaseManager) Delete(req *common.Request, id string) error {
	return fmt.Errorf("Delete resource not implemented")
}

func (m *SResourceBaseManager) IsRawResource() bool {
	return true
}

type SClusterResourceManager struct {
	*SResourceBaseManager
}

func NewClusterResourceManager(keyword, keywordPlural string) *SClusterResourceManager {
	return &SClusterResourceManager{
		SResourceBaseManager: NewResourceBaseManager(keyword, keywordPlural),
	}
}

func (m *SClusterResourceManager) InNamespace() bool {
	return false
}

func (m *SClusterResourceManager) AllowListItems(req *common.Request) bool {
	return req.UserCred.IsSystemAdmin()
}

func (m *SClusterResourceManager) AllowGetItem(req *common.Request, id string) bool {
	return req.UserCred.IsSystemAdmin()
}

func (m *SClusterResourceManager) AllowCreateItem(req *common.Request) bool {
	return req.UserCred.IsSystemAdmin()
}

func (m *SClusterResourceManager) AllowUpdateItem(req *common.Request, id string) bool {
	return req.UserCred.IsSystemAdmin()
}

func (m *SClusterResourceManager) AllowDeleteItem(req *common.Request, id string) bool {
	return req.UserCred.IsSystemAdmin()
}

type SNamespaceResourceManager struct {
	*SResourceBaseManager
}

func NewNamespaceResourceManager(keyword, keywordPlural string) *SNamespaceResourceManager {
	return &SNamespaceResourceManager{
		SResourceBaseManager: NewResourceBaseManager(keyword, keywordPlural),
	}
}

func (m *SNamespaceResourceManager) InNamespace() bool {
	return true
}

func (m *SNamespaceResourceManager) IsOwner(req *common.Request) bool {
	return req.UserCred.IsSystemAdmin() || req.GetDefaultNamespace() == types.ConvertProjectToNamespace(req.UserCred.GetProjectName())
}

func (m *SNamespaceResourceManager) AllowListItems(req *common.Request) bool {
	return m.IsOwner(req)
}

func (m *SNamespaceResourceManager) AllowCreateItem(req *common.Request) bool {
	return m.IsOwner(req)
}

func (m *SNamespaceResourceManager) ValidateCreateData(req *common.Request) error {
	return common.ValidateK8sResourceCreateData(req, apis.TrimKindPlural(m.KeywordPlural()), true)
}

func (m *SNamespaceResourceManager) AllowGetItem(req *common.Request, id string) bool {
	return m.IsOwner(req)
}

func (m *SNamespaceResourceManager) AllowUpdateItem(req *common.Request, id string) bool {
	return m.IsOwner(req)
}

func (m *SNamespaceResourceManager) AllowDeleteItem(req *common.Request, id string) bool {
	return m.IsOwner(req)
}
