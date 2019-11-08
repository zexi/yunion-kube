package rbac

import (
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type SClusterRoleManager struct {
	*resources.SClusterResourceManager
}

type SClusterRoleBindingManager struct {
	*resources.SClusterResourceManager
}

type SRoleManager struct {
	*resources.SNamespaceResourceManager
}

type SRoleBindingManager struct {
	*resources.SNamespaceResourceManager
}

var (
	ClusterRoleManager = &SClusterRoleManager{
		SClusterResourceManager: resources.NewClusterResourceManager("clusterrole", "clusterroles"),
	}
	ClusterRoleBindingManager = &SClusterRoleBindingManager{
		SClusterResourceManager: resources.NewClusterResourceManager("clusterrolebinding", "clusterrolebindings"),
	}
	RoleManager = &SRoleManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("k8s_role", "k8s_roles"),
	}
	RoleBindingManager = &SRoleBindingManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("rolebinding", "rolebindings"),
	}
)

func init() {
	/*for k, man := range map[string]resources.KindManager{
		api.KindNameClusterRole:        ClusterRoleManager,
		api.KindNameClusterRoleBinding: ClusterRoleBinding,
		api.KindNameRole:               RoleManager,
		api.KindNameRoleBinding:        RoleBindingManager,
	} {
		resources.KindManagerMap.Register(k, man)
	}*/
}

func (man *SClusterRoleManager) List(req *common.Request) (common.ListResource, error) {
	return GetClusterRoleList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

func (man *SClusterRoleBindingManager) List(req *common.Request) (common.ListResource, error) {
	return GetClusterRoleBindingList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

func (man *SRoleManager) List(req *common.Request) (common.ListResource, error) {
	return GetRoleList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

func (man *SRoleBindingManager) List(req *common.Request) (common.ListResource, error) {
	return GetRoleBindingList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

type ClusterRoleList struct {
	*common.BaseList
	clusterRoles []*api.ClusterRole
}

func (l *ClusterRoleList) GetResponseData() interface{} {
	return l.clusterRoles
}

func (l *ClusterRoleList) Append(obj interface{}) {
	item := obj.(*rbac.ClusterRole)
	l.clusterRoles = append(l.clusterRoles, ToClusterRole(item, l.GetCluster()))
}

type ClusterRoleBindingList struct {
	*common.BaseList
	bindings []*api.ClusterRoleBinding
}

func (l *ClusterRoleBindingList) GetResponseData() interface{} {
	return l.bindings
}

func (l *ClusterRoleBindingList) Append(obj interface{}) {
	item := obj.(*rbac.ClusterRoleBinding)
	l.bindings = append(l.bindings, ToClusterRoleBinding(item, l.GetCluster()))
}

type RoleList struct {
	*common.BaseList
	roles []*api.Role
}

func (l *RoleList) GetResponseData() interface{} {
	return l.roles
}

func (l *RoleList) Append(obj interface{}) {
	item := obj.(*rbac.Role)
	l.roles = append(l.roles, ToRole(item, l.GetCluster()))
}

type RoleBindingList struct {
	*common.BaseList
	rolebindings []*api.RoleBinding
}

func (l RoleBindingList) GetResponseData() interface{} {
	return l.rolebindings
}

func (l *RoleBindingList) Append(obj interface{}) {
	item := obj.(*rbac.RoleBinding)
	l.rolebindings = append(l.rolebindings, ToRoleBinding(item, l.GetCluster()))
}

func GetClusterRoleList(client *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*ClusterRoleList, error) {
	log.Infof("Getting list of cluster roles")
	items, err := client.ClusterRoleLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	list := &ClusterRoleList{
		BaseList:     common.NewBaseList(cluster),
		clusterRoles: make([]*api.ClusterRole, 0),
	}
	err = dataselect.ToResourceList(
		list,
		items,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	if err != nil {
		return nil, err
	}
	return list, err
}

func GetClusterRoleBindingList(client *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*ClusterRoleBindingList, error) {
	log.Infof("Getting list of cluster bindings")
	items, err := client.ClusterRoleBindingLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	list := &ClusterRoleBindingList{
		BaseList: common.NewBaseList(cluster),
		bindings: make([]*api.ClusterRoleBinding, 0),
	}
	err = dataselect.ToResourceList(
		list,
		items,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	if err != nil {
		return nil, err
	}
	return list, err
}

func GetRoleList(client *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*RoleList, error) {
	log.Infof("Getting list of roles")
	items, err := client.RoleLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	list := &RoleList{
		BaseList: common.NewBaseList(cluster),
		roles:    make([]*api.Role, 0),
	}
	err = dataselect.ToResourceList(
		list,
		items,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	if err != nil {
		return nil, err
	}
	return list, err
}

func GetRoleBindingList(client *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*RoleBindingList, error) {
	log.Infof("Getting list of role bindings")
	items, err := client.RoleBindingLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	list := &RoleBindingList{
		BaseList:     common.NewBaseList(cluster),
		rolebindings: make([]*api.RoleBinding, 0),
	}
	err = dataselect.ToResourceList(
		list,
		items,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	if err != nil {
		return nil, err
	}
	return list, err
}

func ToClusterRole(r *rbac.ClusterRole, cluster api.ICluster) *api.ClusterRole {
	return &api.ClusterRole{
		ObjectMeta:      api.NewObjectMeta(r.ObjectMeta, cluster),
		TypeMeta:        api.NewTypeMeta(r.TypeMeta),
		Rules:           r.Rules,
		AggregationRule: r.AggregationRule,
	}
}

func ToClusterRoleBinding(r *rbac.ClusterRoleBinding, cluster api.ICluster) *api.ClusterRoleBinding {
	return &api.ClusterRoleBinding{
		ObjectMeta: api.NewObjectMeta(r.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(r.TypeMeta),
		Subjects:   r.Subjects,
		RoleRef:    r.RoleRef,
	}
}

func ToRole(r *rbac.Role, cluster api.ICluster) *api.Role {
	return &api.Role{
		ObjectMeta: api.NewObjectMeta(r.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(r.TypeMeta),
		Rules:      r.Rules,
	}
}

func ToRoleBinding(r *rbac.RoleBinding, cluster api.ICluster) *api.RoleBinding {
	return &api.RoleBinding{
		ObjectMeta: api.NewObjectMeta(r.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(r.TypeMeta),
		Subjects:   r.Subjects,
		RoleRef:    r.RoleRef,
	}
}
