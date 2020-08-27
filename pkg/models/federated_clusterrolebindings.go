package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedClusterRoleBindingManager *SFederatedClusterRoleBindingManager
	_                            IFederatedModelManager = new(SFederatedClusterRoleBindingManager)
	_                            IFederatedModel        = new(SFederatedClusterRoleBinding)
)

func init() {
	GetFedClusterRoleBindingManager()
}

func GetFedClusterRoleBindingManager() *SFederatedClusterRoleBindingManager {
	if fedClusterRoleBindingManager == nil {
		fedClusterRoleBindingManager = newModelManager(func() db.IModelManager {
			return &SFederatedClusterRoleBindingManager{
				SFederatedResourceBaseManager: NewFedResourceBaseManager(
					SFederatedClusterRoleBinding{},
					"federatedclusterrolebindings_tbl",
					"federatedclusterrolebinding",
					"federatedclusterrolebindings",
				),
			}
		}).(*SFederatedClusterRoleBindingManager)
	}
	return fedClusterRoleBindingManager
}

// +onecloud:swagger-gen-model-singular=federatedclusterrolebinding
// +onecloud:swagger-gen-model-plural=federatedclusterrolebindings
type SFederatedClusterRoleBindingManager struct {
	SFederatedResourceBaseManager
	SRoleRefResourceBaseManager
}

type SFederatedClusterRoleBinding struct {
	SFederatedResourceBase
	SRoleRefResourceBase
}

func (m *SFederatedClusterRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedClusterRoleBindingCreateInput) (*api.FederatedClusterRoleBindingCreateInput, error) {
	fInput, err := m.SFederatedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedResourceCreateInput = *fInput

	if err := m.SRoleRefResourceBaseManager.ValidateRoleRef(GetClusterRoleManager(), userCred, &input.Spec.Template.RoleRef); err != nil {
		return nil, err
	}

	// TODO: validate cluster rolebinding
	return input, nil
}

func (obj *SFederatedClusterRoleBinding) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := obj.SFederatedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	return nil
}
