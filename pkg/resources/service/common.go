package service

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func ToService(service v1.Service) Service {
	return Service{
		ObjectMeta:        api.NewObjectMeta(service.ObjectMeta),
		TypeMeta:          api.NewTypeMeta(api.ResourceKindService),
		InternalEndpoint:  common.GetInternalEndpoint(service.Name, service.Namespace, service.Spec.Ports),
		ExternalEndpoints: common.GetExternalEndpoints(&service),
		Selector:          service.Spec.Selector,
		ClusterIP:         service.Spec.ClusterIP,
		Type:              service.Spec.Type,
	}
}
