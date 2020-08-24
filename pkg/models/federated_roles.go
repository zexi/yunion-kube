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
	if err := GetRoleManager().ValidateRoleObject(input.ToRole()); err != nil {
		return nil, err
	}
	return input, nil
}
