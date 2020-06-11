package k8smodels

import (
	"reflect"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ServiceManager *SServiceManager
	_              model.IPodOwnerModel = &SService{}
)

func init() {
	ServiceManager = &SServiceManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SService{},
			"k8s_service",
			"k8s_services"),
	}
	ServiceManager.SetVirtualObject(ServiceManager)
}

type SServiceManager struct {
	model.SK8SNamespaceResourceBaseManager
	model.SK8SOwnerResourceBaseManager
}

type SService struct {
	model.SK8SNamespaceResourceBase
}

func (m SServiceManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameService,
		Object:       &v1.Service{},
		KindName:     api.KindNameService,
	}
}

func (m SServiceManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input api.ServiceCreateInput) (runtime.Object, error) {
	objMeta := input.ToObjectMeta()
	return GetServiceFromOption(&objMeta, &input.ServiceCreateOption), nil
}

func (m SServiceManager) GetRawServices(cluster model.ICluster, ns string) ([]*v1.Service, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.ServiceLister().Services(ns).List(labels.Everything())
}

func (m SServiceManager) GetRawServicesByMatchLabels(cluster model.ICluster, ns string, mLabels map[string]string) ([]*v1.Service, error) {
	indexer := cluster.GetHandler().GetIndexer()
	svcs, err := indexer.ServiceLister().Services(ns).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	ret := make([]*v1.Service, 0)
	for _, svc := range svcs {
		if reflect.DeepEqual(svc.Spec.Selector, mLabels) {
			ret = append(ret, svc)
		}
	}
	return ret, nil
}

func (m *SServiceManager) GetAPIServices(cluster model.ICluster, svcs []*v1.Service) ([]*api.Service, error) {
	rets := make([]*api.Service, 0)
	err := ConvertRawToAPIObjects(m, cluster, svcs, &rets)
	return rets, err
}

func (m *SServiceManager) ListItemFilter(ctx *model.RequestContext, q model.IQuery, query *api.ServiceListInput) (model.IQuery, error) {
	q, err := m.SK8SNamespaceResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SNamespaceBase)
	if err != nil {
		return nil, err
	}
	q, err = m.SK8SOwnerResourceBaseManager.ListItemFilter(ctx, q, query.ListInputOwner)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SService) IsOwnerBy(ownerModel model.IK8SModel) (bool, error) {
	return model.IsServiceOwner(ownerModel.(model.IServiceOwnerModel), obj.GetRawService())
}

func (obj *SService) GetRawService() *v1.Service {
	return obj.GetK8SObject().(*v1.Service)
}

func (obj *SService) GetAPIObject() (*api.Service, error) {
	svc := obj.GetRawService()
	return &api.Service{
		ObjectMeta:        obj.GetObjectMeta(),
		TypeMeta:          obj.GetTypeMeta(),
		InternalEndpoint:  GetInternalEndpoint(svc.Name, svc.Namespace, svc.Spec.Ports),
		ExternalEndpoints: GetExternalEndpoints(svc),
		Selector:          svc.Spec.Selector,
		Type:              svc.Spec.Type,
		ClusterIP:         svc.Spec.ClusterIP,
	}, nil
}

func (obj *SService) GetRawPods() ([]*v1.Pod, error) {
	svc := obj.GetRawService()
	selector := labels.SelectorFromSet(svc.Spec.Selector)
	return PodManager.GetRawPodsBySelector(obj.GetCluster(), obj.GetNamespace(), selector)
}

func (obj *SService) GetPods() ([]*api.Pod, error) {
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(obj.GetCluster(), pods)
}

func (obj *SService) GetEvents() ([]*api.Event, error) {
	return EventManager.GetEventsByObject(obj)
}

func (obj *SService) GetAPIDetailObject() (*api.ServiceDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	svc := obj.GetRawService()
	events, err := obj.GetEvents()
	if err != nil {
		return nil, err
	}
	pods, err := obj.GetPods()
	if err != nil {
		return nil, err
	}
	eps, err := EndpointManager.GetAPIEndpointsByService(obj.GetCluster(), svc)
	if err != nil {
		return nil, err
	}
	return &api.ServiceDetail{
		Service:         *apiObj,
		Endpoints:       eps,
		Events:          events,
		Pods:            pods,
		SessionAffinity: svc.Spec.SessionAffinity,
	}, nil
}
