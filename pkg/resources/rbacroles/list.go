package rbacroles

import (
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// RbacRoleList contains a list of Roles and ClusterRoles in the cluster.
type RbacRoleList struct {
	*common.BaseList

	// Unordered list of RbacRoles
	Items []RbacRole `json:"items"`
}

func (l *RbacRoleList) GetResponseData() interface{} {
	return l.Items
}

func (man *SRbacRoleManager) List(req *common.Request) (common.ListResource, error) {
	return GetRbacRoleList(req.GetK8sClient(), req.GetCluster(), req.ToQuery())
}

// RbacRole provides the simplified, combined presentation layer view of Kubernetes' RBAC Roles and ClusterRoles.
// ClusterRoles will be referred to as Roles for the namespace "all namespaces".
type RbacRole struct {
	api.ObjectMeta
	api.TypeMeta
}

// GetRbacRoleList returns a list of all RBAC Roles in the cluster.
func GetRbacRoleList(client kubernetes.Interface, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*RbacRoleList, error) {
	log.Infof("Getting list of RBAC roles")
	channels := &common.ResourceChannels{
		RoleList:        common.GetRoleListChannel(client),
		ClusterRoleList: common.GetClusterRoleListChannel(client),
	}

	return GetRbacRoleListFromChannels(channels, dsQuery, cluster)
}

// GetRbacRoleListFromChannels returns a list of all RBAC roles in the cluster reading required resource list once from the channels.
func GetRbacRoleListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*RbacRoleList, error) {
	roles := <-channels.RoleList.List
	err := <-channels.RoleList.Error
	if err != nil {
		return nil, err
	}

	clusterRoles := <-channels.ClusterRoleList.List
	err = <-channels.ClusterRoleList.Error
	if err != nil {
		return nil, err
	}

	return toRbacRoleLists(roles.Items, clusterRoles.Items, dsQuery, cluster)
}

func toRbacRole(meta v1.ObjectMeta, kind api.ResourceKind, cluster api.ICluster) RbacRole {
	return RbacRole{
		ObjectMeta: api.NewObjectMetaV2(meta, cluster),
		TypeMeta:   api.NewTypeMeta(kind),
	}
}

// toRbacRoleLists merges a list of Roles with a list of ClusterRoles to create a simpler, unified list
func toRbacRoleLists(roles []rbac.Role, clusterRoles []rbac.ClusterRole, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*RbacRoleList, error) {
	result := &RbacRoleList{
		BaseList: common.NewBaseList(cluster),
		Items:    make([]RbacRole, 0),
	}

	err := dataselect.ToResourceList(
		result,
		roles,
		dataselect.NewNamespaceDataCell,
		dsQuery)

	if err != nil {
		return nil, err
	}

	err = dataselect.ToResourceList(
		result,
		clusterRoles,
		dataselect.NewNamespaceDataCell,
		dsQuery)

	return result, err
}

func (l *RbacRoleList) Append(obj interface{}) {
	if item, ok := obj.(rbac.Role); ok {
		l.Items = append(l.Items, toRbacRole(item.ObjectMeta, api.ResourceKindRbacRole, l.GetCluster()))
	} else if item, ok := obj.(rbac.ClusterRole); ok {
		l.Items = append(l.Items, toRbacRole(item.ObjectMeta, api.ResourceKindRbacClusterRole, l.GetCluster()))
	} else {
		log.Errorf("Invalid object for RBAC role: %v", obj)
	}
}
