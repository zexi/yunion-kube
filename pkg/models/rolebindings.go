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
	RoleBindingManager *SRoleBindingManager
	_                  db.IModel = new(SRoleBinding)
)

func init() {
	RoleBindingManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SRoleBindingManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SRoleBinding),
				"rolebindings_tbl",
				"rbacrolebinding",
				"rbacrolebindings",
				api.ResourceNameRoleBinding,
				api.KindNameRoleBinding,
				new(rbac.RoleBinding),
			),
		}
	}).(*SRoleBindingManager)
}

type SRoleBindingManager struct {
	SNamespaceResourceBaseManager
}

type SRoleBinding struct {
	SNamespaceResourceBase
}

func (m *SRoleBindingManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONDict, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support rolebinding create")
}

func (m *SRoleBindingManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	model, err := m.SNamespaceResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func (rb *SRoleBinding) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := rb.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	return nil
}
