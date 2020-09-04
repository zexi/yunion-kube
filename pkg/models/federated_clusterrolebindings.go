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
	// SRoleRefResourceBaseManager
}

type SFederatedClusterRoleBinding struct {
	SFederatedResourceBase
	Spec *api.FederatedClusterRoleBindingSpec `list:"user" update:"user" create:"required"`
	// SRoleRefResourceBase
}

func (m *SFederatedClusterRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedClusterRoleBindingCreateInput) (*api.FederatedClusterRoleBindingCreateInput, error) {
	fInput, err := m.SFederatedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.FederatedResourceCreateInput)
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

func (obj *SFederatedClusterRoleBinding) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := obj.SFederatedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	return nil
}

func (obj *SFederatedClusterRoleBinding) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	_, err := obj.GetManager().PerformAttachCluster(obj, ctx, userCred, data.JSON(data))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFederatedClusterRoleBinding) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	return nil, obj.GetManager().PerformDetachCluster(obj, ctx, userCred, data.JSON(data))
}
