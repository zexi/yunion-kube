package models

import (
	"context"
	"reflect"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
)

var (
	serviceManager *SServiceManager
	_              IPodOwnerModel = new(SService)
)

func init() {
	GetServiceManager()
}

func GetServiceManager() *SServiceManager {
	if serviceManager == nil {
		serviceManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SServiceManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SService{},
					"services_tbl",
					"k8s_service",
					"k8s_services",
					api.ResourceNameService,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameService,
					new(v1.Service),
				),
			}
		}).(*SServiceManager)
	}
	return serviceManager
}

type SServiceManager struct {
	SNamespaceResourceBaseManager
}

type SService struct {
	SNamespaceResourceBase
}

func (m *SServiceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ServiceCreateInputV2) (*api.ServiceCreateInputV2, error) {
	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &data.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.NamespaceResourceCreateInput = *nInput
	return data, nil
}

func (m *SServiceManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.ServiceCreateInputV2)
	data.Unmarshal(input)
	objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, err
	}
	return GetServiceFromOption(&objMeta, &input.ServiceCreateOption), nil
}

func (m *SServiceManager) GetRawServicesByMatchLabels(cli *client.ClusterManager, namespace string, matchLabels map[string]string) ([]*v1.Service, error) {
	indexer := cli.GetHandler().GetIndexer()
	svcs, err := indexer.ServiceLister().Services(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	ret := make([]*v1.Service, 0)
	for _, svc := range svcs {
		if reflect.DeepEqual(svc.Spec.Selector, matchLabels) {
			ret = append(ret, svc)
		}
	}
	return ret, nil
}

func (obj *SService) IsOwnedBy(ownerModel IClusterModel) (bool, error) {
	return IsServiceOwner(ownerModel.(IServiceOwnerModel), obj)
}

func (obj *SService) GetDetails(
	cli *client.ClusterManager,
	base interface{},
	k8sObj runtime.Object,
	isList bool,
) interface{} {
	svc := k8sObj.(*v1.Service)
	detail := api.ServiceDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		InternalEndpoint:        GetInternalEndpoint(svc.Name, svc.Namespace, svc.Spec.Ports),
		ExternalEndpoints:       GetExternalEndpoints(svc),
		Selector:                svc.Spec.Selector,
		Type:                    svc.Spec.Type,
		ClusterIP:               svc.Spec.ClusterIP,
	}
	if isList {
		return detail
	}
	detail.SessionAffinity = svc.Spec.SessionAffinity
	return detail
}

func (obj *SService) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	svc := rawObj.(*v1.Service)
	selector := labels.SelectorFromSet(svc.Spec.Selector)
	return GetPodManager().GetRawPodsBySelector(cli, svc.GetNamespace(), selector)
}
