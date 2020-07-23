package models

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
)

var (
	ServiceManager *SServiceManager
	_              IPodOwnerModel = new(SService)
)

func init() {
	ServiceManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SServiceManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SService),
				"services_tbl",
				"k8s_service",
				"k8s_services",
				api.ResourceNameService,
				api.KindNameService,
				new(v1.Service),
			),
		}
	}).(*SServiceManager)
}

type SServiceManager struct {
	SNamespaceResourceBaseManager
	SK8sOwnedResourceBaseManager
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
	objMeta := input.ToObjectMeta()
	return GetServiceFromOption(&objMeta, &input.ServiceCreateOption), nil
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
	/*
	 * events, err := obj.GetEvents()
	 * if err != nil {
	 *     return nil, err
	 * }
	 * pods, err := obj.GetPods()
	 * if err != nil {
	 *     return nil, err
	 * }
	 */
	detail.SessionAffinity = svc.Spec.SessionAffinity
	return detail
}

func (obj *SService) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	svc := rawObj.(*v1.Service)
	selector := labels.SelectorFromSet(svc.Spec.Selector)
	return PodManager.GetRawPodsBySelector(cli, svc.GetNamespace(), selector)
}
