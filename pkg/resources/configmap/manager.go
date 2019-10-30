package configmap

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/onecloud/pkg/httperrors"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var ConfigMapManager *SConfigMapManager

type SConfigMapManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	ConfigMapManager = &SConfigMapManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("configmap", "configmaps"),
	}
}

func (man *SConfigMapManager) ValidateCreateData(req *common.Request) error {
	return man.SNamespaceResourceManager.ValidateCreateData(req)
}

func (man *SConfigMapManager) Create(req *common.Request) (interface{}, error) {
	input := &api.ConfigMapCreateInput{}
	if err := req.Data.Unmarshal(input); err != nil {
		return nil, err
	}
	if len(input.Data) == 0 {
		return nil, httperrors.NewNotAcceptableError("data is empty")
	}
	objMeta, err := common.GetK8sObjectCreateMetaByRequest(req)
	if err != nil {
		return nil, err
	}
	cfg := &v1.ConfigMap{
		ObjectMeta: *objMeta,
		Data:       input.Data,
	}
	return req.GetK8sClient().CoreV1().ConfigMaps(objMeta.GetNamespace()).Create(cfg)
}
