package service

import (
	api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	return CreateService(req, option)
}

type CreateOption struct {
	ObjectMeta metaV1.ObjectMeta
	apis.ServiceCreateOption
}

func (o CreateOption) ToService() *api.Service {
	return common.GetServiceFromOption(&o.ObjectMeta, &o.ServiceCreateOption)
}

func CreateService(req *common.Request, opt CreateOption) (*api.Service, error) {
	svc := opt.ToService()
	return common.CreateService(req, svc)
}
