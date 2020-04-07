package model

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/yunion-kube/pkg/apis"
)

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

func (m *SK8SClusterResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query apis.ListInputK8SClusterBase) (IQuery, error) {
	if query.Name != "" {
		q.AddFilter(func(obj metav1.Object) bool {
			return obj.GetName() != query.Name
		})
	}
	return m.SK8SModelBaseManager.ListItemFilter(ctx, q, query.ListInputK8SBase)
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

func (m *SK8SNamespaceResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query apis.ListInputK8SNamespaceBase) (IQuery, error) {
	if query.Namespace != "" {
		q.Namespace(query.Namespace)
		/*q.AddFilter(func(obj metav1.Object) bool {
			return obj.GetNamespace() != query.Namespace
		})*/
	}
	return m.SK8SClusterResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SClusterBase)
}

func (m SK8SNamespaceResourceBaseManager) ValidateCreateData(
	ctx *RequestContext, _ *jsonutils.JSONDict,
	input *apis.K8sNamespaceResourceCreateInput) (*apis.K8sNamespaceResourceCreateInput, error) {
	// TODO: check namespace resource exists
	return input, nil
}

func (m SK8SNamespaceResourceBase) GetNamespace() string {
	return m.GetMetaObject().GetNamespace()
}
