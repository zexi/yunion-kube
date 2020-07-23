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
	ClusterRoleManager *SClusterRoleManager
	_                  db.IModel = new(SClusterRole)
)

func init() {
	ClusterRoleManager = NewK8sModelManager(func() ISyncableManager {
		return &SClusterRoleManager{
			SClusterResourceBaseManager: NewClusterResourceBaseManager(
				new(SClusterRole),
				"clusterroles_tbl",
				"rbacclusterrole",
				"rbacclusterroles",
				api.ResourceNameClusterRole,
				api.KindNameClusterRole,
				new(rbac.ClusterRole),
			),
		}
	}).(*SClusterRoleManager)
}

type SClusterRoleManager struct {
	SClusterResourceBaseManager
}

type SClusterRole struct {
	SClusterResourceBase
}

func (m *SClusterRoleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
}

func (m *SClusterRoleManager) SyncResources(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error {
	return SyncClusterResources(ctx, userCred, cluster, m)
}

func (m *SClusterRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONDict, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support clusterrole create")
}

func (m *SClusterRoleManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	model, err := m.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func (cr *SClusterRole) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := cr.SClusterResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	return nil
}

func (cr *SClusterRole) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	cr.SClusterResourceBase.PostDelete(ctx, userCred)
}
