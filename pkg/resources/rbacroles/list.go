package rbacroles

import (
	"k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

// RbacRoleList contains a list of Roles and ClusterRoles in the cluster.
type RbacRoleList struct {
	*common.BaseList

	// Unordered list of RbacRoles
	Items []*api.RbacRole
}

type RbacRoleBindingList struct {
	*common.BaseList
	Items []*api.RbacRoleBinding
}

func (l *RbacRoleList) GetResponseData() interface{} {
	return l.Items
}

func (l *RbacRoleBindingList) GetResponseData() interface{} {
	return l.Items
}

func (man *SRbacRoleManager) List(req *common.Request) (common.ListResource, error) {
	return GetRbacRoleList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

func (man *SRbacRoleBindingManager) List(req *common.Request) (common.ListResource, error) {
	return GetRbacRoleBindingList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

// GetRbacRoleList returns a list of all RBAC Roles in the cluster.
func GetRbacRoleList(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*RbacRoleList, error) {
	log.Infof("Getting list of RBAC roles")
	channels := &common.ResourceChannels{
		RoleList:        common.GetRoleListChannel(indexer),
		ClusterRoleList: common.GetClusterRoleListChannel(indexer),
	}

	return GetRbacRoleListFromChannels(channels, dsQuery, cluster)
}

func GetRbacRoleBindingList(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*RbacRoleBindingList, error) {
	rbs, err := indexer.RoleBindingLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	crbs, err := indexer.ClusterRoleBindingLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	return toRbacRoleBindingLists(rbs, crbs, dsQuery, cluster)
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

	return toRbacRoleLists(roles, clusterRoles, dsQuery, cluster)
}

func toRbacRole(obj *rbac.Role, cluster api.ICluster) *api.RbacRole {
	return &api.RbacRole{
		ObjectMeta: api.NewObjectMeta(obj.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(obj.TypeMeta),
		Type:       TypeRole,
		Rules:      obj.Rules,
	}
}

func toRbacRoleBinding(obj *rbac.RoleBinding, cluster api.ICluster) *api.RbacRoleBinding {
	return &api.RbacRoleBinding{
		ObjectMeta: api.NewObjectMeta(obj.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(obj.TypeMeta),
		Type:       TypeRoleBinding,
		Subjects:   obj.Subjects,
		RoleRef:    obj.RoleRef,
	}
}

func toRbacClusterRole(obj *rbac.ClusterRole, cluster api.ICluster) *api.RbacRole {
	return &api.RbacRole{
		ObjectMeta:      api.NewObjectMeta(obj.ObjectMeta, cluster),
		TypeMeta:        api.NewTypeMeta(obj.TypeMeta),
		Type:            TypeClusterRole,
		Rules:           obj.Rules,
		AggregationRule: obj.AggregationRule,
	}
}

func toRbacClusterRoleBinding(obj *rbac.ClusterRoleBinding, cluster api.ICluster) *api.RbacRoleBinding {
	return &api.RbacRoleBinding{
		ObjectMeta: api.NewObjectMeta(obj.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(obj.TypeMeta),
		Type:       TypeClusterRoleBinding,
		Subjects:   obj.Subjects,
		RoleRef:    obj.RoleRef,
	}
}

// toRbacRoleLists merges a list of Roles with a list of ClusterRoles to create a simpler, unified list
func toRbacRoleLists(roles []*rbac.Role, clusterRoles []*rbac.ClusterRole, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*RbacRoleList, error) {
	result := &RbacRoleList{
		BaseList: common.NewBaseList(cluster),
		Items:    make([]*api.RbacRole, 0),
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
	if item, ok := obj.(*rbac.Role); ok {
		l.Items = append(l.Items, toRbacRole(item, l.GetCluster()))
	} else if item, ok := obj.(*rbac.ClusterRole); ok {
		l.Items = append(l.Items, toRbacClusterRole(item, l.GetCluster()))
	} else {
		log.Errorf("Invalid object for RBAC role: %v", obj)
	}
}

func toRbacRoleBindingLists(rbs []*rbac.RoleBinding, crbs []*rbac.ClusterRoleBinding, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*RbacRoleBindingList, error) {
	result := &RbacRoleBindingList{
		BaseList: common.NewBaseList(cluster),
		Items:    make([]*api.RbacRoleBinding, 0),
	}
	err := dataselect.ToResourceList(
		result,
		rbs,
		dataselect.NewNamespaceDataCell,
		dsQuery)

	if err != nil {
		return nil, err
	}

	err = dataselect.ToResourceList(
		result,
		crbs,
		dataselect.NewNamespaceDataCell,
		dsQuery)

	return result, err
}

func (l *RbacRoleBindingList) Append(obj interface{}) {
	if item, ok := obj.(*rbac.RoleBinding); ok {
		l.Items = append(l.Items, toRbacRoleBinding(item, l.GetCluster()))
	} else if item, ok := obj.(*rbac.ClusterRoleBinding); ok {
		l.Items = append(l.Items, toRbacClusterRoleBinding(item, l.GetCluster()))
	} else {
		log.Errorf("Invalid object for RBAC role: %v", obj)
	}
}

func toSA(obj *v1.ServiceAccount, cluster api.ICluster) *api.ServiceAccount {
	return &api.ServiceAccount{
		TypeMeta:                     api.NewTypeMeta(obj.TypeMeta),
		ObjectMeta:                   api.NewObjectMeta(obj.ObjectMeta, cluster),
		Secrets:                      obj.Secrets,
		ImagePullSecrets:             obj.ImagePullSecrets,
		AutomountServiceAccountToken: obj.AutomountServiceAccountToken,
	}
}

func (man *SServiceAccountManager) List(req *common.Request) (common.ListResource, error) {
	return GetSAList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

func GetSAList(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*SAList, error) {
	sas, err := indexer.ServiceAccountLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := &SAList{
		BaseList: common.NewBaseList(cluster),
		Items:    make([]*api.ServiceAccount, 0),
	}
	if err := dataselect.ToResourceList(
		result,
		sas,
		dataselect.NewNamespaceDataCell,
		dsQuery); err != nil {
		return nil, err
	}
	return result, err
}

type SAList struct {
	*common.BaseList
	Items []*api.ServiceAccount
}

func (l *SAList) GetResponseData() interface{} {
	return l.Items
}

func (l *SAList) Append(obj interface{}) {
	item, ok := obj.(*v1.ServiceAccount)
	if !ok {
		log.Errorf("Invalid object for ServiceAccount: %v", obj)
	}
	l.Items = append(l.Items, toSA(item, l.GetCluster()))
}
