package service

import (
	api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SServiceManager) ValidateCreateData(req *common.Request) error {
	return man.SNamespaceResourceManager.ValidateCreateData(req)
}

func (man *SServiceManager) Create(req *common.Request) (interface{}, error) {
	objMeta, err := common.GetK8sObjectCreateMetaByRequest(req)
	if err != nil {
		return nil, err
	}
	opt := &apis.ServiceCreateOption{}
	if err := req.Data.Unmarshal(opt); err != nil {
		return nil, err
	}
	svcType := opt.Type
	if svcType == "" {
		svcType = string(api.ServiceTypeClusterIP)
	}
	if opt.IsExternal {
		svcType = string(api.ServiceTypeLoadBalancer)
	}
	if !utils.IsInStringArray(svcType, []string{string(api.ServiceTypeClusterIP), string(api.ServiceTypeLoadBalancer)}) {
		return nil, httperrors.NewInputParameterError("service type %s not supported", svcType)
	}
	opt.Type = svcType
	if len(opt.Selector) == 0 {
		return nil, httperrors.NewInputParameterError("Selector is empty")
	}
	option := CreateOption{
		ObjectMeta:          *objMeta,
		ServiceCreateOption: *opt,
	}
	return CreateService(req.GetK8sClient(), option)
}

type CreateOption struct {
	ObjectMeta metaV1.ObjectMeta
	apis.ServiceCreateOption
}

func (o CreateOption) ToService() *api.Service {
	svc := &api.Service{
		ObjectMeta: o.ObjectMeta,
		Spec: api.ServiceSpec{
			Selector: o.Selector,
			Type:     api.ServiceType(o.Type),
		},
	}
	if o.LoadBalancerNetwork != "" {
		svc.Annotations = map[string]string{
			common.YUNION_LB_NETWORK_ANNOTATION: o.LoadBalancerNetwork,
		}
	}
	svc.Spec.Ports = GetServicePorts(o.PortMappings)
	return svc
}

func CreateService(cli client.Interface, opt CreateOption) (*api.Service, error) {
	svc := opt.ToService()
	return cli.CoreV1().Services(opt.ObjectMeta.GetNamespace()).Create(svc)
}

func GetServicePorts(ps []apis.PortMapping) []api.ServicePort {
	ports := []api.ServicePort{}
	for _, p := range ps {
		ports = append(ports, p.ToServicePort())
	}
	return ports
}
