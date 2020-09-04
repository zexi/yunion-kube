package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedRoleManager *SFederatedRoleManager
	_              IFederatedModelManager = new(SFederatedRoleManager)
	_              IFederatedModel        = new(SFederatedRole)
)

func init() {
	GetFedRoleManager()
}

func GetFedRoleManager() *SFederatedRoleManager {
	if fedRoleManager == nil {
		fedRoleManager = newModelManager(func() db.IModelManager {
			return &SFederatedRoleManager{
				SFederatedNamespaceResourceManager: NewFedNamespaceResourceManager(
					SFederatedRole{},
					"federatedroles_tbl",
					"federatedrole",
					"federatedroles",
				),
			}
		}).(*SFederatedRoleManager)
	}
	return fedRoleManager
}

// +onecloud:swagger-gen-model-singular=federatedrole
// +onecloud:swagger-gen-model-plural=federatedroles
type SFederatedRoleManager struct {
	SFederatedNamespaceResourceManager
}

type SFederatedRole struct {
	SFederatedNamespaceResource
	Spec *api.FederatedRoleSpec `list:"user" update:"user" create:"required"`
}

func (m *SFederatedRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedRoleCreateInput) (*api.FederatedRoleCreateInput, error) {
	nInput, err := m.SFederatedNamespaceResourceManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedNamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedNamespaceResourceCreateInput = *nInput
	if err := GetRoleManager().ValidateRoleObject(input.ToRole(nInput.Federatednamespace)); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SFederatedRoleManager) GetPropertyApiResources(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterAPIGroupResources, error) {
	ret, err := GetFedClustersApiResources(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterAPIGroupResources), nil
}

func (m *SFederatedRoleManager) GetPropertyClusterUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUsers, error) {
	ret, err := GetFedClustersUsers(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterUsers), nil
}

func (m *SFederatedRoleManager) GetPropertyClusterUserGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUserGroups, error) {
	ret, err := GetFedClustersUserGroups(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return ret.(api.ClusterUserGroups), nil
}

func (obj *SFederatedRole) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	_, err := obj.GetManager().PerformAttachCluster(obj, ctx, userCred, data.JSON(data))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFederatedRole) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	return nil, obj.GetManager().PerformDetachCluster(obj, ctx, userCred, data.JSON(data))
}
