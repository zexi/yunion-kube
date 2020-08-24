package models

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/apis/rbac/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
)

var (
	clusterRoleManager *SClusterRoleManager
	_                  IClusterModelManager = new(SClusterRoleManager)
	_                  IClusterModel        = new(SClusterRole)
)

func init() {
	GetClusterRoleManager()
}

func GetClusterRoleManager() *SClusterRoleManager {
	if clusterRoleManager == nil {
		clusterRoleManager = NewK8sModelManager(func() ISyncableManager {
			return &SClusterRoleManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					new(SClusterRole),
					"clusterroles_tbl",
					"rbacclusterrole",
					"rbacclusterroles",
					api.ResourceNameClusterRole,
					api.KindNameClusterRole,
					new(rbacv1.ClusterRole),
				),
			}
		}).(*SClusterRoleManager)
	}
	return clusterRoleManager
}

type SClusterRoleManager struct {
	SClusterResourceBaseManager
}

type SClusterRole struct {
	SClusterResourceBase
}

func (m *SClusterRoleManager) GetRoleKind() string {
	return api.KindNameClusterRole
}

func (m *SClusterRoleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
}

func (m *SClusterRoleManager) SyncResources(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error {
	return SyncClusterResources(ctx, userCred, cluster, m)
}

func (m *SClusterRoleManager) ValidateClusterRoleObject(obj *rbacv1.ClusterRole) error {
	return ValidateK8sObject(obj, new(rbac.ClusterRole), func(out interface{}) field.ErrorList {
		return validation.ValidateClusterRole(out.(*rbac.ClusterRole))
	})
}

func (m *SClusterRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ClusterRoleCreateInput) (*api.ClusterRoleCreateInput, error) {
	cInput, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.ClusterResourceCreateInput = *cInput
	cRole := input.ToClusterRole()
	if err := m.ValidateClusterRoleObject(cRole); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SClusterRoleManager) NewRemoteObjectForCreate(_ IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.ClusterRoleCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	return input.ToClusterRole(), nil
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

func (cr *SClusterRole) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := cr.SClusterResourceBase.GetDetails(cli, base, k8sObj, isList).(api.ClusterResourceDetail)
	role := k8sObj.(*rbacv1.ClusterRole)
	out := api.ClusterRoleDetail{
		ClusterResourceDetail: detail,
		Rules:                 role.Rules,
		AggregationRule:       role.AggregationRule,
	}
	return out
}
