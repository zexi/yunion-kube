package models

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
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
	roleManager *SRoleManager
	_           IClusterModel = new(SRole)
)

func init() {
	GetRoleManager()
}

func GetRoleManager() *SRoleManager {
	if roleManager == nil {
		roleManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SRoleManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					&SRole{},
					"roles_tbl",
					"rbacrole",
					"rbacroles",
					api.ResourceNameRole,
					api.KindNameRole,
					new(rbacv1.Role),
				),
			}
		}).(*SRoleManager)
	}
	return roleManager
}

type SRoleManager struct {
	SNamespaceResourceBaseManager
}

type SRole struct {
	SNamespaceResourceBase
}

func (m *SRoleManager) GetRoleKind() string {
	return api.KindNameRole
}

func (m *SRoleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SRoleManager) ValidateRoleObject(role *rbacv1.Role) error {
	return ValidateK8sObject(role, new(rbac.Role), func(out interface{}) field.ErrorList {
		return validation.ValidateRole(out.(*rbac.Role))
	})
}

func (m *SRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.RoleCreateInput) (*api.RoleCreateInput, error) {
	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.NamespaceResourceCreateInput = *nInput
	role := input.ToRole()
	if err := m.ValidateRoleObject(role); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SRoleManager) NewRemoteObjectForCreate(_ IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.RoleCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	return input.ToRole(), nil
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
