package k8smodels

import (
	v1 "k8s.io/api/core/v1"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ServiceAccountManager *SServiceAccountManager
)

func init() {
	ServiceAccountManager = &SServiceAccountManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			new(SServiceAccount), "serviceaccount", "serviceaccounts"),
	}
	ServiceAccountManager.SetVirtualObject(ServiceAccountManager)
}

type SServiceAccountManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SServiceAccount struct {
	model.SK8SNamespaceResourceBase
}

func (m *SServiceAccountManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameServiceAccount,
		Object:       new(v1.ServiceAccount),
		KindName:     api.KindNameServiceAccount,
	}
}

func (obj *SServiceAccount) GetRawServiceAccount() *v1.ServiceAccount {
	return obj.GetK8SObject().(*v1.ServiceAccount)
}

func (obj *SServiceAccount) GetAPIObject() (*api.ServiceAccount, error) {
	sa := obj.GetRawServiceAccount()
	return &api.ServiceAccount{
		ObjectMeta:                   obj.GetObjectMeta(),
		TypeMeta:                     obj.GetTypeMeta(),
		Secrets:                      sa.Secrets,
		ImagePullSecrets:             sa.ImagePullSecrets,
		AutomountServiceAccountToken: sa.AutomountServiceAccountToken,
	}, nil
}

func (obj *SServiceAccount) GetAPIDetailObject() (*api.ServiceAccount, error) {
	return obj.GetAPIObject()
}
