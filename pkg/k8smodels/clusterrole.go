package k8smodels

import (
	rbac "k8s.io/api/rbac/v1"
	"strings"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ClusterRoleManager *SClusterRoleManager
)

func init() {
	ClusterRoleManager = &SClusterRoleManager{
		SK8SClusterResourceBaseManager: model.NewK8SClusterResourceBaseManager(
			new(SClusterRole),
			"rbacclusterrole",
			"rbacclusterroles"),
	}
	ClusterRoleManager.SetVirtualObject(ClusterRoleManager)
}

type SClusterRoleManager struct {
	model.SK8SClusterResourceBaseManager
}

type SClusterRole struct {
	model.SK8SClusterResourceBase
}

func (_ SClusterRoleManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameClusterRole,
		Object:       &rbac.ClusterRole{},
		KindName:     apis.KindNameClusterRole,
	}
}

func (obj SClusterRole) GetRawClusterRole() *rbac.ClusterRole {
	return obj.GetK8SObject().(*rbac.ClusterRole)
}

func (obj SClusterRole) GetType() string {
	return strings.ToLower(apis.KindNameClusterRole)
}

func (obj SClusterRole) GetAPIObject() (*apis.RbacRole, error) {
	cr := obj.GetRawClusterRole()
	return &apis.RbacRole{
		ObjectMeta:      obj.GetObjectMeta(),
		TypeMeta:        obj.GetTypeMeta(),
		Type:            obj.GetType(),
		Rules:           cr.Rules,
		AggregationRule: cr.AggregationRule,
	}, nil
}

func (obj SClusterRole) GetAPIDetailObject() (*apis.RbacRole, error) {
	return obj.GetAPIObject()
}
