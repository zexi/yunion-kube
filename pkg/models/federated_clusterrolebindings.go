package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedClusterRoleBindingManager *SFedClusterRoleBindingManager
	_                            IFedModelManager = new(SFedClusterRoleBindingManager)
	_                            IFedModel        = new(SFedClusterRoleBinding)
)

func init() {
	GetFedClusterRoleBindingManager()
}

func GetFedClusterRoleBindingManager() *SFedClusterRoleBindingManager {
	if fedClusterRoleBindingManager == nil {
		fedClusterRoleBindingManager = newModelManager(func() db.IModelManager {
			return &SFedClusterRoleBindingManager{
				SFedResourceBaseManager: NewFedResourceBaseManager(
					SFedClusterRoleBinding{},
					"federatedclusterrolebindings_tbl",
					"federatedclusterrolebinding",
					"federatedclusterrolebindings",
				),
			}
		}).(*SFedClusterRoleBindingManager)
	}
	return fedClusterRoleBindingManager
}

// +onecloud:swagger-gen-model-singular=federatedclusterrolebinding
// +onecloud:swagger-gen-model-plural=federatedclusterrolebindings
type SFedClusterRoleBindingManager struct {
	SFedResourceBaseManager
	// SRoleRefResourceBaseManager
}

type SFedClusterRoleBinding struct {
	SFedResourceBase
	Spec *api.FederatedClusterRoleBindingSpec `list:"user" update:"user" create:"required"`
	// SRoleRefResourceBase
}

func (m *SFedClusterRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedClusterRoleBindingCreateInput) (*api.FederatedClusterRoleBindingCreateInput, error) {
	fInput, err := m.SFedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	log.Errorf("====input %#v", input.Spec.Template)
	input.FederatedResourceCreateInput = *fInput
	if err := ValidateFederatedRoleRef(ctx, userCred, input.Spec.Template.RoleRef); err != nil {
		return nil, err
	}
	crb := input.ToClusterRoleBinding()
	if err := GetClusterRoleBindingManager().ValidateClusterRoleBinding(crb); err != nil {
		return nil, err
	}
	return input, nil
}

func (obj *SFedClusterRoleBinding) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := obj.SFedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	return nil
}
