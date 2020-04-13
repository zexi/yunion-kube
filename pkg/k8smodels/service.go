package k8smodels

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ServiceManager *SServiceManager
	_              model.IK8SModel = &SService{}
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
}

type SService struct {
	model.SK8SNamespaceResourceBase
}

func (m SServiceManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameService,
		Object:       &v1.Service{},
	}
}

func (m SServiceManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input apis.ServiceCreateInput) (runtime.Object, error) {
	return nil, nil
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

func (m *SServiceManager) GetAPIServices(cluster model.ICluster, svcs []*v1.Service) ([]*apis.Service, error) {
	rets := make([]*apis.Service, 0)
	err := ConvertRawToAPIObjects(m, cluster, svcs, &rets)
	return rets, err
}

func (obj *SService) GetRawService() *v1.Service {
	return obj.GetK8SObject().(*v1.Service)
}

func (obj *SService) GetAPIObject() (*apis.Service, error) {
	svc := obj.GetRawService()
	return &apis.Service{
		ObjectMeta:        obj.GetObjectMeta(),
		TypeMeta:          obj.GetTypeMeta(),
		InternalEndpoint:  GetInternalEndpoint(svc.Name, svc.Namespace, svc.Spec.Ports),
		ExternalEndpoints: GetExternalEndpoints(svc),
		Selector:          svc.Spec.Selector,
		Type:              svc.Spec.Type,
		ClusterIP:         svc.Spec.ClusterIP,
	}, nil
}

func (obj *SService) GetPods() ([]*apis.Pod, error) {
	svc := obj.GetRawService()
	selector := labels.SelectorFromSet(svc.Spec.Selector)
	pods, err := PodManager.GetRawPodsBySelector(obj.GetCluster(), obj.GetNamespace(), selector)
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(obj.GetCluster(), pods)
}

func (obj *SService) GetEvents() ([]*apis.Event, error) {
	return EventManager.GetEventsByObject(obj)
}

func (obj *SService) GetAPIDetailObject() (*apis.ServiceDetail, error) {
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
	return &apis.ServiceDetail{
		Service:         *apiObj,
		Endpoints:       eps,
		Events:          events,
		Pods:            pods,
		SessionAffinity: svc.Spec.SessionAffinity,
	}, nil
}
