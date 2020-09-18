package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	FedRoleClusterManager *SFedRoleClusterManager
	_                     IFedJointClusterModel = new(SFedRoleCluster)
)

func init() {
	db.InitManager(func() {
		FedRoleClusterManager = NewFedJointManager(func() db.IJointModelManager {
			return &SFedRoleClusterManager{
				SFedNamespaceJointClusterManager: NewFedNamespaceJointClusterManager(
					SFedRoleCluster{},
					"federatedroleclusters_tbl",
					"federatedrolecluster",
					"federatedroleclusters",
					GetFedRoleManager(),
					GetRoleManager(),
				),
			}
		}).(*SFedRoleClusterManager)
		GetFedRoleManager().SetJointModelManager(FedRoleClusterManager)
		RegisterFedJointClusterManager(GetFedRoleManager(), FedRoleClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedrolecluster
// +onecloud:swagger-gen-model-plural=federatedroleclusters
type SFedRoleClusterManager struct {
	SFedNamespaceJointClusterManager
}

type SFedRoleCluster struct {
	SFederatedNamespaceJointCluster
}

func (obj *SFedRoleCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedRoleCluster) GetFedRole() (*SFedRole, error) {
	fedObj, err := GetFedResAPI().JointResAPI().FetchFedResourceModel(obj)
	if err != nil {
		return nil, err
	}
	return fedObj.(*SFedRole), nil
}

func (obj *SFedRoleCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, base api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj, err := obj.GetFedRole()
	if err != nil {
		return nil, errors.Wrapf(err, "get federated role object")
	}
	input := api.RoleCreateInput{
		NamespaceResourceCreateInput: base,
		Rules:                        fedObj.Spec.Template.Rules,
	}
	return input.JSON(input), nil
}
