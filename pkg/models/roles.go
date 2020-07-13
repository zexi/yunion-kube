package models

import (
	"context"

	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	RoleManager *SRoleManager
	_           IClusterModel = new(SRole)
)

func init() {
	RoleManager = NewK8sModelManager(func() IClusterModelManager {
		return &SRoleManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				&SRole{},
				"roles_tbl",
				"rbacrole",
				"rbacroles",
				api.ResourceNameRole,
				api.KindNameRole,
				new(rbac.Role),
			),
		}
	}).(*SRoleManager)
}

type SRoleManager struct {
	SNamespaceResourceBaseManager
}

type SRole struct {
	SNamespaceResourceBase
}

func (m *SRoleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONDict, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support role create")
}

func (m *SRoleManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	model, err := m.SNamespaceResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	//kRole := obj.(*rbac.Role)
	//roleObj := model.(*SRole)
	return model, nil
}

func (r *SRole) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := r.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	return nil
}

func (r *SRole) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	r.SNamespaceResourceBase.PostDeleteV2(r, ctx, userCred)
}
