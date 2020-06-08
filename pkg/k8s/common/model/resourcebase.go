package model

import (
	"reflect"
	"strings"

	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/api"
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

func (m *SK8SClusterResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query api.ListInputK8SClusterBase) (IQuery, error) {
	if query.Name != "" {
		q.AddFilter(func(obj IK8SModel) (bool, error) {
			return obj.GetName() == query.Name || strings.Contains(obj.GetName(), query.Name), nil
		})
	}
	return m.SK8SModelBaseManager.ListItemFilter(ctx, q, query.ListInputK8SBase)
}

func (m *SK8SClusterResourceBaseManager) ValidateCreateData(
	ctx *RequestContext,
	_ *jsonutils.JSONDict,
	input *api.K8sClusterResourceCreateInput) (*api.K8sClusterResourceCreateInput, error) {
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

func (m *SK8SNamespaceResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query api.ListInputK8SNamespaceBase) (IQuery, error) {
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
	input *api.K8sNamespaceResourceCreateInput) (*api.K8sNamespaceResourceCreateInput, error) {
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

type SK8SOwnerResourceBaseManager struct{}

type IK8SOwnerResource interface {
	IsOwnerBy(ownerModel IK8SModel) (bool, error)
}

func (m SK8SOwnerResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query api.ListInputOwner) (IQuery, error) {
	if !query.ShouldDo() {
		return q, nil
	}
	q.AddFilter(m.ListOwnerFilter(query))
	return q, nil
}

func (m SK8SOwnerResourceBaseManager) ListOwnerFilter(query api.ListInputOwner) QueryFilter {
	return func(obj IK8SModel) (bool, error) {
		man := GetK8SModelManagerByKind(query.OwnerKind)
		if man == nil {
			return false, httperrors.NewNotFoundError("Not found owner_kind %s", query.OwnerKind)
		}
		ownerModel, err := NewK8SModelObjectByName(man, obj.GetCluster(), obj.GetNamespace(), query.OwnerName)
		if err != nil {
			return false, err
		}
		return obj.(IK8SOwnerResource).IsOwnerBy(ownerModel)
	}
}

func IsPodOwner(model IPodOwnerModel, pod *v1.Pod) (bool, error) {
	pods, err := model.GetRawPods()
	if err != nil {
		return false, err
	}
	return IsObjectContains(pod, pods), nil
}

func IsServiceOwner(model IServiceOwnerModel, svc *v1.Service) (bool, error) {
	svcs, err := model.GetRawServices()
	if err != nil {
		return false, err
	}
	return IsObjectContains(svc, svcs), nil
}

func IsObjectContains(obj metav1.Object, objs interface{}) bool {
	objsV := reflect.ValueOf(objs)
	for i := 0; i < objsV.Len(); i++ {
		ov := objsV.Index(i).Interface().(metav1.Object)
		if obj.GetUID() == ov.GetUID() {
			return true
		}
	}
	return false
}

func IsEventOwner(model IK8SModel, event *v1.Event) (bool, error) {
	metaObj := model.GetObjectMeta()
	return event.InvolvedObject.UID == metaObj.GetUID(), nil
}

func IsJobOwner(model IK8SModel, job *batch.Job) (bool, error) {
	metaObj := model.GetObjectMeta()
	for _, i := range job.OwnerReferences {
		if i.UID == metaObj.GetUID() {
			return true, nil
		}
	}
	return false, nil
}
