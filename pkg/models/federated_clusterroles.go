package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedClusterRoleManager *SFedClusterRoleManager
	_                     IFedModelManager = new(SFedClusterRoleManager)
	_                     IFedModel        = new(SFedClusterRole)
)

// +onecloud:swagger-gen-model-singular=federatedclusterrole
// +onecloud:swagger-gen-model-plural=federatedclusterroles
type SFedClusterRoleManager struct {
	SFedResourceBaseManager
}

type SFedClusterRole struct {
	SFedResourceBase
	Spec *api.FederatedClusterRoleSpec `list:"user" update:"user" create:"required"`
}

func init() {
	GetFedClusterRoleManager()
}

func GetFedClusterRoleManager() *SFedClusterRoleManager {
	if fedClusterRoleManager == nil {
		fedClusterRoleManager = newModelManager(func() db.IModelManager {
			return &SFedClusterRoleManager{
				SFedResourceBaseManager: NewFedResourceBaseManager(
					SFedClusterRole{},
					"federatedclusterroles_tbl",
					"federatedclusterrole",
					"federatedclusterroles",
				),
			}
		}).(*SFedClusterRoleManager)
	}
	return fedClusterRoleManager
}

func (m *SFedClusterRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedClusterRoleCreateInput) (*api.FederatedClusterRoleCreateInput, error) {
	cInput, err := m.SFedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedResourceCreateInput = *cInput
	if err := GetClusterRoleManager().ValidateClusterRoleObject(input.ToClusterRole()); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SFedClusterRoleManager) GetFedClusterRole(id string) (*SFedClusterRole, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SFedClusterRole), nil
}

func (m *SFedClusterRoleManager) GetPropertyApiResources(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterAPIGroupResources, error) {
	ret, err := GetFedClustersApiResources(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterAPIGroupResources), nil
}

func (m *SFedClusterRoleManager) GetPropertyClusterUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUsers, error) {
	ret, err := GetFedClustersUsers(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterUsers), nil
}

func (m *SFedClusterRoleManager) GetPropertyClusterUserGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUserGroups, error) {
	ret, err := GetFedClustersUserGroups(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterUserGroups), nil
}

func (obj *SFedClusterRole) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	_, err := obj.SFedResourceBase.PerformAttachCluster(ctx, userCred, query, data.JSON(data))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFedClusterRole) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	_, err := obj.SFedResourceBase.PerformDetachCluster(ctx, userCred, query, data.JSON(data))
	return nil, err
}
