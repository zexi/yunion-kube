package resources

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/resources/common"
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
	log.Fatalf("AllowListItems not implemented")
	return false
}

func (m *SResourceBaseManager) List(req *common.Request) (*modules.ListResult, error) {
	log.Fatalf("List not implemented")
	return nil, nil
}

func (m *SResourceBaseManager) Get(req *common.Request, id string) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Get resource not implemented")
}

func (m *SResourceBaseManager) ValidateCreateData(req *common.Request) error {
	return nil
}

func (m *SResourceBaseManager) Create(req *common.Request) (jsonutils.JSONObject, error) {
	log.Fatalf("Create not implemented")
	return nil, nil
}

func (m *SResourceBaseManager) Delete(req *common.Request, id string) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Delete resource not implemented")
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
