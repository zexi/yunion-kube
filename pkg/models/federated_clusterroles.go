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
	Spec *api.FederatedRoleSpec `list:"user" update:"user" create:"required"`
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
