package resources

import (
	"github.com/yunionio/log"
	"k8s.io/client-go/kubernetes"
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

func (m *SResourceBaseManager) AllowListItems(req *Request) bool {
	log.Fatalf("AllowListItems not implemented")
	return false
}

func (m *SResourceBaseManager) List(k8sCli kubernetes.Interface, req *Request) {
	log.Fatalf("List not implemented")
}
