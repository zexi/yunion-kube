package resources

import (
	"k8s.io/client-go/kubernetes"
	"yunion.io/x/log"

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

func (m *SResourceBaseManager) AllowListItems(req *common.Request) bool {
	log.Fatalf("AllowListItems not implemented")
	return false
}

func (m *SResourceBaseManager) List(k8sCli kubernetes.Interface, req *common.Request) {
	log.Fatalf("List not implemented")
}
