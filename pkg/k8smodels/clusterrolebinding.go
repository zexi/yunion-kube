package k8smodels

import (
	"strings"

	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ClusterRoleBindingManager *SClusterRoleBindingManager
)

func init() {
	ClusterRoleBindingManager = &SClusterRoleBindingManager{
		SK8SClusterResourceBaseManager: model.NewK8SClusterResourceBaseManager(
			new(SRoleBinding),
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
		ResourceName: apis.ResourceNameClusterRoleBinding,
		Object:       new(rbac.ClusterRoleBinding),
	}
}

func (obj *SClusterRoleBinding) GetRawRoleBinding() *rbac.ClusterRoleBinding {
	return obj.GetK8SObject().(*rbac.ClusterRoleBinding)
}

func (obj *SClusterRoleBinding) GetType() string {
	return strings.ToLower(apis.KindNameClusterRoleBinding)
}

func (obj *SClusterRoleBinding) GetAPIObject() (*apis.RbacRoleBinding, error) {
	rb := obj.GetRawRoleBinding()
	return &apis.RbacRoleBinding{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Type:       obj.GetType(),
		Subjects:   rb.Subjects,
		RoleRef:    rb.RoleRef,
	}, nil
}

func (obj *SClusterRoleBinding) GetAPIDetailObject() (*apis.RbacRoleBinding, error) {
	return obj.GetAPIObject()
}
