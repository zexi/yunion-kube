package models

import (
	"context"

	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	ClusterRoleBindingManager *SClusterRoleBindingManager
	_                         db.IModel = new(SClusterRoleBinding)
)

func init() {
	ClusterRoleBindingManager = NewK8sModelManager(func() ISyncableManager {
		return &SClusterRoleBindingManager{
			SClusterResourceBaseManager: NewClusterResourceBaseManager(
				new(SClusterRoleBinding),
				"clusterrolebindings_tbl",
				"rbacclusterrolebinding",
				"rbacclusterrolebindings",
				api.ResourceNameClusterRoleBinding,
				api.KindNameClusterRoleBinding,
				new(rbac.ClusterRoleBinding),
			),
		}
	}).(*SClusterRoleBindingManager)
}

type SClusterRoleBindingManager struct {
	SClusterResourceBaseManager
}

type SClusterRoleBinding struct {
	SClusterResourceBase
}

func (m *SClusterRoleBindingManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
}

func (m *SClusterRoleBindingManager) SyncResources(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error {
	return SyncClusterResources(ctx, userCred, cluster, m)
}

func (m *SClusterRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONDict, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support clusterrolebinding create")
}

func (m *SClusterRoleBindingManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	return m.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
}

func (crb *SClusterRoleBinding) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	return crb.SClusterResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj)
}

func (crb *SClusterRoleBinding) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	crb.SClusterResourceBase.PostDelete(ctx, userCred)
}
