package k8smodels

import (
	"strings"

	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/yunion-kube/pkg/api"
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
		ResourceName: api.ResourceNameRoleBinding,
		Object:       &rbac.RoleBinding{},
		KindName:     api.KindNameRoleBinding,
	}
}

func (obj *SRoleBinding) GetRawRoleBinding() *rbac.RoleBinding {
	return obj.GetK8SObject().(*rbac.RoleBinding)
}

func (obj *SRoleBinding) GetType() string {
	return strings.ToLower(api.KindNameRoleBinding)
}

func (obj *SRoleBinding) GetAPIObject() (*api.RbacRoleBinding, error) {
	rb := obj.GetRawRoleBinding()
	return &api.RbacRoleBinding{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Type:       obj.GetType(),
		Subjects:   rb.Subjects,
		RoleRef:    rb.RoleRef,
	}, nil
}

func (obj *SRoleBinding) GetAPIDetailObject() (*api.RbacRoleBinding, error) {
	return obj.GetAPIObject()
}
