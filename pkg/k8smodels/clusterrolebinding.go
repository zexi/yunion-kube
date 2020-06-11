package k8smodels

import (
	"strings"

	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ClusterRoleBindingManager *SClusterRoleBindingManager
)

func init() {
	ClusterRoleBindingManager = &SClusterRoleBindingManager{
		SK8SClusterResourceBaseManager: model.NewK8SClusterResourceBaseManager(
			new(SClusterRoleBinding),
			"rbacclusterrolebinding",
			"rbacclusterrolebindings"),
	}
	ClusterRoleBindingManager.SetVirtualObject(ClusterRoleBindingManager)
}

type SClusterRoleBindingManager struct {
	model.SK8SClusterResourceBaseManager
}

type SClusterRoleBinding struct {
	model.SK8SClusterResourceBase
}

func (m *SClusterRoleBindingManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameClusterRoleBinding,
		Object:       &rbac.ClusterRoleBinding{},
		KindName:     api.KindNameClusterRoleBinding,
	}
}

func (obj *SClusterRoleBinding) GetRawRoleBinding() *rbac.ClusterRoleBinding {
	return obj.GetK8SObject().(*rbac.ClusterRoleBinding)
}

func (obj *SClusterRoleBinding) GetType() string {
	return strings.ToLower(api.KindNameClusterRoleBinding)
}

func (obj *SClusterRoleBinding) GetAPIObject() (*api.RbacRoleBinding, error) {
	rb := obj.GetRawRoleBinding()
	return &api.RbacRoleBinding{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Type:       obj.GetType(),
		Subjects:   rb.Subjects,
		RoleRef:    rb.RoleRef,
	}, nil
}

func (obj *SClusterRoleBinding) GetAPIDetailObject() (*api.RbacRoleBinding, error) {
	return obj.GetAPIObject()
}
