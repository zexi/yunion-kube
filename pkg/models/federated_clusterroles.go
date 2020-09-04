package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedClusterRoleManager *SFederatedClusterRoleManager
	_                     IFederatedModelManager = new(SFederatedClusterRoleManager)
	_                     IFederatedModel        = new(SFederatedClusterRole)
)

// +onecloud:swagger-gen-model-singular=federatedclusterrole
// +onecloud:swagger-gen-model-plural=federatedclusterroles
type SFederatedClusterRoleManager struct {
	SFederatedResourceBaseManager
}

type SFederatedClusterRole struct {
	SFederatedResourceBase
	Spec *api.FederatedClusterRoleSpec `list:"user" update:"user" create:"required"`
}

func init() {
	GetFedClusterRoleManager()
}

func GetFedClusterRoleManager() *SFederatedClusterRoleManager {
	if fedClusterRoleManager == nil {
		fedClusterRoleManager = newModelManager(func() db.IModelManager {
			return &SFederatedClusterRoleManager{
				SFederatedResourceBaseManager: NewFedResourceBaseManager(
					SFederatedClusterRole{},
					"federatedclusterroles_tbl",
					"federatedclusterrole",
					"federatedclusterroles",
				),
			}
		}).(*SFederatedClusterRoleManager)
	}
	return fedClusterRoleManager
}

func (m *SFederatedClusterRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedClusterRoleCreateInput) (*api.FederatedClusterRoleCreateInput, error) {
	cInput, err := m.SFederatedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedResourceCreateInput = *cInput
	if err := GetClusterRoleManager().ValidateClusterRoleObject(input.ToClusterRole()); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SFederatedClusterRoleManager) GetFedClusterRole(id string) (*SFederatedClusterRole, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SFederatedClusterRole), nil
}

func (m *SFederatedClusterRoleManager) GetPropertyApiResources(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterAPIGroupResources, error) {
	ret, err := GetFedClustersApiResources(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterAPIGroupResources), nil
}

func (m *SFederatedClusterRoleManager) GetPropertyClusterUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUsers, error) {
	ret, err := GetFedClustersUsers(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterUsers), nil
}

func (m *SFederatedClusterRoleManager) GetPropertyClusterUserGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUserGroups, error) {
	ret, err := GetFedClustersUserGroups(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterUserGroups), nil
}

func (obj *SFederatedClusterRole) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	_, err := obj.GetManager().PerformAttachCluster(obj, ctx, userCred, data.JSON(data))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFederatedClusterRole) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	return nil, obj.GetManager().PerformDetachCluster(obj, ctx, userCred, data.JSON(data))
}
