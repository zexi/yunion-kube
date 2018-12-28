package service

import (
	api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SServiceManager) ValidateCreateData(req *common.Request) error {
	return man.SNamespaceResourceManager.ValidateCreateData(req)
}

func (man *SServiceManager) Create(req *common.Request) (interface{}, error) {
	objMeta, err := common.GetK8sObjectCreateMeta(req.Data)
	if err != nil {
		return nil, err
	}
	data := req.Data
	svcType, _ := data.GetString("type")
	if svcType == "" {
		svcType = string(api.ServiceTypeClusterIP)
	}
	if isExternal, _ := data.Bool("isExternal"); isExternal {
		svcType = string(api.ServiceTypeLoadBalancer)
	}
	if !utils.IsInStringArray(svcType, []string{string(api.ServiceTypeClusterIP), string(api.ServiceTypeLoadBalancer)}) {
		return nil, httperrors.NewInputParameterError("service type %s not supported", svcType)
	}
	portMaps := []PortMapping{}
	portMapsObj, err := req.Data.Get("portMappings")
	if err != nil {
		return nil, httperrors.NewInputParameterError("No ports spec")
	}
	err = portMapsObj.Unmarshal(&portMaps)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Invalid ports input: %v", err)
	}
	selector := make(map[string]string)
	data.Unmarshal(&selector, "selector")
	if len(selector) == 0 {
		return nil, httperrors.NewInputParameterError("Selector is empty")
	}
	opt := CreateOption{
		ObjectMeta: *objMeta,
		Ports:      portMaps,
		Type:       api.ServiceType(svcType),
		Selector:   selector,
		Namespace:  req.GetDefaultNamespace(),
	}
	if lbNet, _ := data.GetString("loadBalancerNetwork"); lbNet != "" {
		opt.LBNetwork = lbNet
	}
	return CreateService(req.GetK8sClient(), opt)
}

type CreateOption struct {
	ObjectMeta metaV1.ObjectMeta
	Ports      []PortMapping
	Type       api.ServiceType
	LBNetwork  string
	Selector   map[string]string
	Namespace  string
}

func (o CreateOption) ToService() *api.Service {
	svc := &api.Service{
		ObjectMeta: o.ObjectMeta,
		Spec: api.ServiceSpec{
			Selector: o.Selector,
			Type:     o.Type,
		},
	}
	if o.LBNetwork != "" {
		svc.Annotations = map[string]string{
			common.YUNION_LB_NETWORK_ANNOTATION: o.LBNetwork,
		}
	}
	svc.Spec.Ports = GetServicePorts(o.Ports)
	return svc
}

func CreateService(cli client.Interface, opt CreateOption) (*api.Service, error) {
	svc := opt.ToService()
	return cli.CoreV1().Services(opt.Namespace).Create(svc)
}

func GetServicePorts(ps []PortMapping) []api.ServicePort {
	ports := []api.ServicePort{}
	for _, p := range ps {
		ports = append(ports, p.ToServicePort())
	}
	return ports
}
