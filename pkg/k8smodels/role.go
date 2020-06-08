package k8smodels

import (
	"strings"

	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	RoleManager *SRoleManager
)

func init() {
	RoleManager = &SRoleManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			new(SRole),
			"rbacrole",
			"rbacroles"),
	}
	RoleManager.SetVirtualObject(RoleManager)
}

type SRoleManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SRole struct {
	model.SK8SNamespaceResourceBase
}

func (_ SRoleManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameRole,
		Object:       &rbac.Role{},
		KindName:     api.KindNameRole,
	}
}

func (obj SRole) GetType() string {
	return strings.ToLower(api.KindNameRole)
}

func (obj SRole) GetRawRole() *rbac.Role {
	return obj.GetK8SObject().(*rbac.Role)
}

func (obj SRole) GetAPIObject() (*api.RbacRole, error) {
	cr := obj.GetRawRole()
	return &api.RbacRole{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Type:       obj.GetType(),
		Rules:      cr.Rules,
	}, nil
}

func (obj SRole) GetAPIDetailObject() (*api.RbacRole, error) {
	return obj.GetAPIObject()
}
