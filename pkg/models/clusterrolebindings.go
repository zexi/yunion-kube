package models

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/apis/rbac/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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
	SRoleRefResourceBaseManager
}

type SClusterRoleBinding struct {
	SClusterResourceBase
	SRoleRefResourceBase
}

func (m *SClusterRoleBindingManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
}

func (m *SClusterRoleBindingManager) SyncResources(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error {
	return SyncClusterResources(ctx, userCred, cluster, m)
}

func (m *SClusterRoleBindingManager) ValidateClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) error {
	return ValidateK8sObject(crb, new(rbac.ClusterRoleBinding), func(out interface{}) field.ErrorList {
		return validation.ValidateClusterRoleBinding(out.(*rbac.ClusterRoleBinding))
	})
}

func (m *SClusterRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ClusterRoleBindingCreateInput) (*api.ClusterRoleBindingCreateInput, error) {
	cInput, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.ClusterResourceCreateInput = *cInput

	if err := m.SRoleRefResourceBaseManager.ValidateRoleRef(GetClusterRoleManager(), userCred, input.RoleRef); err != nil {
		return nil, err
	}

	crb := input.ToClusterRoleBinding()
	if err := m.ValidateClusterRoleBinding(crb); err != nil {
		return nil, err
	}
	return input, nil
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
