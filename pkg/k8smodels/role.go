package k8smodels

import (
	"strings"

	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/yunion-kube/pkg/apis"
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
		ResourceName: apis.ResourceNameRole,
		Object:       &rbac.Role{},
		KindName:     apis.KindNameRole,
	}
}

func (obj SRole) GetType() string {
	return strings.ToLower(apis.KindNameRole)
}

func (obj SRole) GetRawRole() *rbac.Role {
	return obj.GetK8SObject().(*rbac.Role)
}

func (obj SRole) GetAPIObject() (*apis.RbacRole, error) {
	cr := obj.GetRawRole()
	return &apis.RbacRole{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Type:       obj.GetType(),
		Rules:      cr.Rules,
	}, nil
}

func (obj SRole) GetAPIDetailObject() (*apis.RbacRole, error) {
	return obj.GetAPIObject()
}
