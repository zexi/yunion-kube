package service

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func ToService(service *v1.Service, cluster api.ICluster) api.Service {
	return api.Service{
		ObjectMeta:        api.NewObjectMeta(service.ObjectMeta, cluster),
		TypeMeta:          api.NewTypeMeta(service.TypeMeta),
		InternalEndpoint:  common.GetInternalEndpoint(service.Name, service.Namespace, service.Spec.Ports),
		ExternalEndpoints: common.GetExternalEndpoints(service),
		Selector:          service.Spec.Selector,
		ClusterIP:         service.Spec.ClusterIP,
		Type:              service.Spec.Type,
	}
}
