package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedRoleBindingManager *SFedRoleBindingManager
	_                     IFedModelManager = new(SFedRoleBindingManager)
	_                     IFedModel        = new(SFedRoleBinding)
)

func init() {
	GetFedRoleBindingManager()
}

func GetFedRoleBindingManager() *SFedRoleBindingManager {
	if fedRoleBindingManager == nil {
		fedRoleBindingManager = newModelManager(func() db.IModelManager {
			return &SFedRoleBindingManager{
				SFedNamespaceResourceManager: NewFedNamespaceResourceManager(
					SFedRoleBinding{},
					"federatedrolebindings_tbl",
					"federatedrolebinding",
					"federatedrolebindings",
				),
			}
		}).(*SFedRoleBindingManager)
	}
	return fedRoleBindingManager
}

// +onecloud:swagger-gen-model-singular=federatedrolebinding
// +onecloud:swagger-gen-model-plural=federatedrolebindings
type SFedRoleBindingManager struct {
	SFedNamespaceResourceManager
	// SRoleRefResourceBase
}

type SFedRoleBinding struct {
	SFedNamespaceResource
	Spec *api.FederatedRoleBindingSpec `list:"user" update:"user" create:"required"`
	// SRoleRefResourceBase
}

func (m *SFedRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedRoleBindingCreateInput) (*api.FederatedRoleBindingCreateInput, error) {
	nInput, err := m.SFedNamespaceResourceManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedNamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedNamespaceResourceCreateInput = *nInput
	if err := ValidateFederatedRoleRef(ctx, userCred, input.Spec.Template.RoleRef); err != nil {
		return nil, err
	}
	if err := GetRoleBindingManager().ValidateRoleBinding(input.ToRoleBinding(nInput.Federatednamespace)); err != nil {
		return nil, err
	}
	return input, nil
}
