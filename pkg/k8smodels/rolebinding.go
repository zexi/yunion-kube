package k8smodels

import (
	"strings"

	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	RoleBindingManager *SRoleBindingManager
)

func init() {
	RoleBindingManager = &SRoleBindingManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			new(SRoleBinding),
			"rbacrolebinding",
			"rbacrolebindings"),
	}
	RoleBindingManager.SetVirtualObject(RoleBindingManager)
}

type SRoleBindingManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SRoleBinding struct {
	model.SK8SNamespaceResourceBase
}

func (m *SRoleBindingManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameRoleBinding,
		Object:       new(rbac.RoleBinding),
	}
}

func (obj *SRoleBinding) GetRawRoleBinding() *rbac.RoleBinding {
	return obj.GetK8SObject().(*rbac.RoleBinding)
}

func (obj *SRoleBinding) GetType() string {
	return strings.ToLower(apis.KindNameRoleBinding)
}

func (obj *SRoleBinding) GetAPIObject() (*apis.RbacRoleBinding, error) {
	rb := obj.GetRawRoleBinding()
	return &apis.RbacRoleBinding{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Type:       obj.GetType(),
		Subjects:   rb.Subjects,
		RoleRef:    rb.RoleRef,
	}, nil
}

func (obj *SRoleBinding) GetAPIDetailObject() (*apis.RbacRoleBinding, error) {
	return obj.GetAPIObject()
}
