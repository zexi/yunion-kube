package k8smodels

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	NamespaceManager *SNamespaceManager
	_                model.IK8SModel = &SNamespace{}
)

func init() {
	NamespaceManager = &SNamespaceManager{
		SK8SClusterResourceBaseManager: model.NewK8SClusterResourceBaseManager(
			&SNamespace{},
			"namespace",
			"namespaces"),
	}
	NamespaceManager.SetVirtualObject(NamespaceManager)
}

type SNamespaceManager struct {
	model.SK8SClusterResourceBaseManager
}

type SNamespace struct {
	model.SK8SClusterResourceBase
}

func (m SNamespaceManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameNamespace,
		Object:       &v1.Namespace{},
		KindName:     api.KindNameNamespace,
	}
}

func (m SNamespaceManager) GetRawNamespaces(cluster model.ICluster) ([]*v1.Namespace, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.NamespaceLister().List(labels.Everything())
}

func (man *SNamespaceManager) ValidateCreateData(
	ctx *model.RequestContext,
	query *jsonutils.JSONDict,
	input *api.NamespaceCreateInput) (*api.NamespaceCreateInput, error) {
	if _, err := man.SK8SClusterResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sClusterResourceCreateInput); err != nil {
		return nil, err
	}
	return input, nil
}

func (man *SNamespaceManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input api.NamespaceCreateInput) (runtime.Object, error) {
	objMeta := input.ToObjectMeta()
	ns := &v1.Namespace{
		ObjectMeta: objMeta,
	}
	return ns, nil
}

func (obj *SNamespace) GetRawNamespace() *v1.Namespace {
	return obj.GetK8SObject().(*v1.Namespace)
}

func (obj *SNamespace) GetAPIObject() (*api.Namespace, error) {
	ns := obj.GetRawNamespace()
	return &api.Namespace{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Phase:      ns.Status.Phase,
	}, nil
}

func (obj *SNamespace) GetEvents() ([]*api.Event, error) {
	return EventManager.GetNamespaceEvents(obj.GetCluster(), obj.GetName())
}

func (obj *SNamespace) GetResourceQuotas() ([]*api.ResourceQuotaDetail, error) {
	return ResourceQuotaManager.GetResourceQuotaDetails(obj.GetCluster(), obj.GetName())
}

func (obj *SNamespace) GetResourceLimits() ([]*api.LimitRange, error) {
	return LimitRangeManager.GetLimitRanges(obj.GetCluster(), obj.GetName())
}

func (obj *SNamespace) GetAPIDetailObject() (*api.NamespaceDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	events, err := obj.GetEvents()
	if err != nil {
		return nil, err
	}
	rsQuotas, err := obj.GetResourceQuotas()
	if err != nil {
		return nil, err
	}
	limitRanges, err := obj.GetResourceLimits()
	if err != nil {
		return nil, err
	}
	return &api.NamespaceDetail{
		Namespace:      *apiObj,
		Events:         events,
		ResourceQuotas: rsQuotas,
		ResourceLimits: limitRanges,
	}, nil
}
