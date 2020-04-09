package model

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
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
	m := SK8SClusterResourceBaseManager{
		NewK8SModelBaseManager(dt, keyword, keywordPlural),
	}
	m.RegisterOrderFields(
		OrderFieldCreationTimestamp{},
		OrderFieldName())
	return m
}

func (m *SK8SClusterResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query apis.ListInputK8SClusterBase) (IQuery, error) {
	if query.Name != "" {
		q.AddFilter(func(obj IK8SModel) bool {
			return obj.GetName() == query.Name
		})
	}
	return m.SK8SModelBaseManager.ListItemFilter(ctx, q, query.ListInputK8SBase)
}

func (m *SK8SClusterResourceBaseManager) ValidateCreateData(
	ctx *RequestContext,
	_ *jsonutils.JSONDict,
	input *apis.K8sClusterResourceCreateInput) (*apis.K8sClusterResourceCreateInput, error) {
	if input.Cluster == "" {
		return nil, httperrors.NewNotEmptyError("cluster is empty")
	}
	return input, nil
}

func (m *SK8SClusterResourceBase) ValidateUpdateData(
	_ *RequestContext, query, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotAcceptableError("%s not support update", m.Keyword())
}

func (m *SK8SClusterResourceBase) ValidateDeleteCondition(
	_ *RequestContext, _, _ *jsonutils.JSONDict) error {
	return nil
}

func (m *SK8SClusterResourceBase) CustomizeDelete(
	ctx *RequestContext, _, _ *jsonutils.JSONDict) error {
	return nil
}

func (m *SK8SClusterResourceBase) GetName() string {
	return m.GetObjectMeta().GetName()
}

type SK8SNamespaceResourceBase struct {
	SK8SClusterResourceBase
}

type SK8SNamespaceResourceBaseManager struct {
	SK8SClusterResourceBaseManager
}

func NewK8SNamespaceResourceBaseManager(dt interface{}, keyword string, keywordPlural string) SK8SNamespaceResourceBaseManager {
	man := SK8SNamespaceResourceBaseManager{NewK8SClusterResourceBaseManager(dt, keyword, keywordPlural)}
	man.RegisterOrderFields(
		OrderFieldNamespace(),
		OrderFieldStatus())
	return man
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
	ctx *RequestContext, query *jsonutils.JSONDict,
	input *apis.K8sNamespaceResourceCreateInput) (*apis.K8sNamespaceResourceCreateInput, error) {
	cInput, err := m.SK8SClusterResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	// TODO: check namespace resource exists
	input.K8sClusterResourceCreateInput = *cInput
	return input, nil
}

func (m SK8SNamespaceResourceBase) GetNamespace() string {
	return m.GetMetaObject().GetNamespace()
}
