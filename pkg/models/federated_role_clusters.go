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
	_                     IFederatedJointClusterModel = new(SFedRoleCluster)
)

func init() {
	db.InitManager(func() {
		FedRoleClusterManager = NewFederatedJointManager(func() db.IJointModelManager {
			return &SFedRoleClusterManager{
				SFederatedNamespaceJointClusterManager: NewFedNamespaceJointClusterManager(
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
	SFederatedNamespaceJointClusterManager
}

type SFedRoleCluster struct {
	SFederatedNamespaceJointCluster
}

func (obj *SFedRoleCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedRoleCluster) GetFedRole() (*SFederatedRole, error) {
	fedObj, err := obj.GetFedResourceModel()
	if err != nil {
		return nil, err
	}
	return fedObj.(*SFederatedRole), nil
}

func (obj *SFedRoleCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, base api.ClusterResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj, err := obj.GetFedRole()
	if err != nil {
		return nil, errors.Wrapf(err, "get federated role object")
	}
	nsBase, err := obj.SFederatedNamespaceJointCluster.GetResourceCreateInput(userCred, base)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster namespace base create input")
	}
	input := api.RoleCreateInput{
		NamespaceResourceCreateInput: nsBase,
		Rules:                        fedObj.Spec.Template.Rules,
	}
	return input.JSON(input), nil
}
