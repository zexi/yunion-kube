package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedRoleBindingManager *SFederatedRoleBindingManager
	_                     IFederatedModelManager = new(SFederatedRoleBindingManager)
	_                     IFederatedModel        = new(SFederatedRoleBinding)
)

func init() {
	GetFedRoleBindingManager()
}

func GetFedRoleBindingManager() *SFederatedRoleBindingManager {
	if fedRoleBindingManager == nil {
		fedRoleBindingManager = newModelManager(func() db.IModelManager {
			return &SFederatedRoleBindingManager{
				SFederatedNamespaceResourceManager: NewFedNamespaceResourceManager(
					SFederatedRoleBinding{},
					"federatedrolebindings_tbl",
					"federatedrolebinding",
					"federatedrolebindings",
				),
			}
		}).(*SFederatedRoleBindingManager)
	}
	return fedRoleBindingManager
}

// +onecloud:swagger-gen-model-singular=federatedrolebinding
// +onecloud:swagger-gen-model-plural=federatedrolebindings
type SFederatedRoleBindingManager struct {
	SFederatedNamespaceResourceManager
	// SRoleRefResourceBase
}

type SFederatedRoleBinding struct {
	SFederatedNamespaceResource
	Spec *api.FederatedRoleBindingSpec `list:"user" update:"user" create:"required"`
	// SRoleRefResourceBase
}

func (m *SFederatedRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedRoleBindingCreateInput) (*api.FederatedRoleBindingCreateInput, error) {
	nInput, err := m.SFederatedNamespaceResourceManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedNamespaceResourceCreateInput)
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

func (obj *SFederatedRoleBinding) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	_, err := obj.GetManager().PerformAttachCluster(obj, ctx, userCred, data.JSON(data))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFederatedRoleBinding) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	return nil, obj.GetManager().PerformDetachCluster(obj, ctx, userCred, data.JSON(data))
}
