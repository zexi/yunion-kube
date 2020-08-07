package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedClusterRoleManager *SFederatedClusterRoleManager
	_                     IFederatedModelManager = new(SFederatedClusterRoleManager)
	_                     IFederatedModel        = new(SFederatedClusterRole)
)

type SFederatedClusterRoleManager struct {
	SFederatedResourceBaseManager
}

type SFederatedClusterRole struct {
	SFederatedResourceBase
	Spec *api.FederatedRoleSpec `list:"user" update:"user" create:"required"`
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
