package model

type SK8SClusterResourceBase struct {
	SK8SModelBase

	Cluster string `json:"cluster"`
}

type SK8SClusterResourceBaseManager struct {
	SK8SModelBaseManager
}

func NewK8SClusterResourceBaseManager(dt interface{}, keyword, keywordPlural string) SK8SClusterResourceBaseManager {
	return SK8SClusterResourceBaseManager{
		NewK8SModelBaseManager(dt, keyword, keywordPlural),
	}
}

type SK8SNamespaceResourceBase struct {
	SK8SClusterResourceBase
}

type SK8SNamespaceResourceBaseManager struct {
	SK8SClusterResourceBaseManager
}

func NewK8SNamespaceResourceBaseManager(dt interface{}, keyword string, keywordPlural string) SK8SNamespaceResourceBaseManager {
	return SK8SNamespaceResourceBaseManager{NewK8SClusterResourceBaseManager(dt, keyword, keywordPlural)}
}

func (m SK8SNamespaceResourceBase) GetNamespace() string {
	return m.GetMetaObject().GetNamespace()
}
