package rbacroles

import (
	"strings"

	"yunion.io/x/onecloud/pkg/httperrors"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var (
	RbacRoleManager        *SRbacRoleManager
	RbacRoleBindingManager *SRbacRoleBindingManager
	ServiceAccountManager  *SServiceAccountManager

	TypeRole               = strings.ToLower(api.KindNameRole)
	TypeRoleBinding        = strings.ToLower(api.KindNameRoleBinding)
	TypeClusterRole        = strings.ToLower(api.KindNameClusterRole)
	TypeClusterRoleBinding = strings.ToLower(api.KindNameClusterRoleBinding)
)

type SRbacRoleManager struct {
	*resources.SNamespaceResourceManager
}

type SRbacRoleBindingManager struct {
	*resources.SNamespaceResourceManager
}

type SServiceAccountManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	RbacRoleManager = &SRbacRoleManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("rbacrole", "rbacroles"),
	}
	RbacRoleBindingManager = &SRbacRoleBindingManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("rbacrolebinding", "rbacrolebindings"),
	}
	ServiceAccountManager = &SServiceAccountManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("serviceaccount", "serviceaccounts"),
	}

	for k, man := range map[string]resources.KindManager{
		api.KindNameClusterRole:        RbacRoleManager.ToClusterRoleManager(),
		api.KindNameClusterRoleBinding: RbacRoleBindingManager.ToClusterRoleBindingManager(),
		api.KindNameRole:               RbacRoleManager.ToRoleManager(),
		api.KindNameRoleBinding:        RbacRoleBindingManager.ToRoleBindingManager(),
		api.KindNameServiceAccount:     ServiceAccountManager,
	} {
		resources.KindManagerMap.Register(k, man)
	}
}

type clusterRoleManager struct {
	*SRbacRoleManager
}

type roleManager struct {
	*SRbacRoleManager
}

type clusterRoleBindingManager struct {
	*SRbacRoleBindingManager
}

type roleBindingManager struct {
	*SRbacRoleBindingManager
}

func (man *SRbacRoleManager) ToClusterRoleManager() *clusterRoleManager {
	return &clusterRoleManager{
		SRbacRoleManager: man,
	}
}

func (man *SRbacRoleManager) ToRoleManager() *roleManager {
	return &roleManager{
		SRbacRoleManager: man,
	}
}

func (man *SRbacRoleBindingManager) ToClusterRoleBindingManager() *clusterRoleBindingManager {
	return &clusterRoleBindingManager{
		SRbacRoleBindingManager: man,
	}
}

func (man *SRbacRoleBindingManager) ToRoleBindingManager() *roleBindingManager {
	return &roleBindingManager{
		SRbacRoleBindingManager: man,
	}
}

func (man *clusterRoleManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetClusterRoleDetails(cli, cluster, name)
}

func (man *clusterRoleBindingManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetClusterRoleBindingDetails(cli, cluster, name)
}

func (man *roleManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetRoleDetails(cli, cluster, namespace, name)
}

func (man *roleBindingManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetRoleBindingDetails(cli, cluster, namespace, name)
}

func (man *SRbacRoleManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	roleType, _ := req.Query.GetString("type")
	if roleType == "" {
		roleType = TypeRole
	}
	return GetRbacRoleDetails(req.GetIndexer(), req.GetCluster(), roleType, namespace, id)
}

func (man *SRbacRoleBindingManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	bindingType, _ := req.Query.GetString("type")
	if bindingType == "" {
		bindingType = TypeRoleBinding
	}
	return GetRbacRoleBindingDetails(req.GetIndexer(), req.GetCluster(), bindingType, namespace, id)
}

func (man *SServiceAccountManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetSADetails(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery().ToRequestParam(), id)
}

func (man *SServiceAccountManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetSADetails(cli, cluster, namespace, name)
}

func GetSADetails(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*api.ServiceAccount, error) {
	obj, err := indexer.ServiceAccountLister().ServiceAccounts(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return toSA(obj, cluster), nil
}

func GetRbacRoleDetails(indexer *client.CacheFactory, cluster api.ICluster, roleType, namespace, name string) (*api.RbacRole, error) {
	switch roleType {
	case TypeClusterRole:
		return GetClusterRoleDetails(indexer, cluster, name)
	case TypeRole:
		return GetRoleDetails(indexer, cluster, namespace, name)
	}
	return nil, httperrors.NewInputParameterError("unsupported role type: %s", roleType)
}

func GetRbacRoleBindingDetails(indexer *client.CacheFactory, cluster api.ICluster, bType, namespace, name string) (*api.RbacRoleBinding, error) {
	switch bType {
	case TypeClusterRoleBinding:
		return GetClusterRoleBindingDetails(indexer, cluster, name)
	case TypeRoleBinding:
		return GetRoleBindingDetails(indexer, cluster, namespace, name)
	}
	return nil, httperrors.NewInputParameterError("unsupported role type: %s", bType)
}

func GetRoleDetails(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*api.RbacRole, error) {
	obj, err := indexer.RoleLister().Roles(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return toRbacRole(obj, cluster), nil
}

func GetRoleBindingDetails(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*api.RbacRoleBinding, error) {
	obj, err := indexer.RoleBindingLister().RoleBindings(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return toRbacRoleBinding(obj, cluster), nil
}

func GetClusterRoleDetails(indexer *client.CacheFactory, cluster api.ICluster, name string) (*api.RbacRole, error) {
	obj, err := indexer.ClusterRoleLister().Get(name)
	if err != nil {
		return nil, err
	}
	return toRbacClusterRole(obj, cluster), nil
}

func GetClusterRoleBindingDetails(indexer *client.CacheFactory, cluster api.ICluster, name string) (*api.RbacRoleBinding, error) {
	obj, err := indexer.ClusterRoleBindingLister().Get(name)
	if err != nil {
		return nil, err
	}
	return toRbacClusterRoleBinding(obj, cluster), nil
}
