package ingress

import (
	extensions "k8s.io/api/extensions/v1beta1"

	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SIngressManager) ValidateCreateData(req *common.Request) error {
	return man.SNamespaceResourceManager.ValidateCreateData(req)
}

func (man *SIngressManager) Create(req *common.Request) (interface{}, error) {
	spec := extensions.IngressSpec{}
	err := common.JsonDecode(req.Data, &spec)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	cli := req.GetK8sClient()
	namespace := req.GetDefaultNamespace()
	objMeta, err := common.GetK8sObjectCreateMeta(req.Data)
	if err != nil {
		return nil, err
	}
	obj := &extensions.Ingress{
		ObjectMeta: *objMeta,
		Spec:       spec,
	}
	ing, err := cli.Extensions().Ingresses(namespace).Create(obj)
	return ing, err
}
