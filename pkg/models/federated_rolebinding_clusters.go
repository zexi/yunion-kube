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
	FedRoleBindingClusterManager *SFedRoleBindingClusterManager
	_                            IFederatedJointClusterModel = new(SFedRoleBindingCluster)
)

func init() {
	db.InitManager(func() {
		FedRoleBindingClusterManager = NewFederatedJointManager(func() db.IJointModelManager {
			return &SFedRoleBindingClusterManager{
				SFederatedNamespaceJointClusterManager: NewFedNamespaceJointClusterManager(
					SFedRoleBindingCluster{},
					"federatedrolebindingclusters_tbl",
					"federatedrolebindingcluster",
					"federatedrolebindingclusters",
					GetFedRoleBindingManager(),
					GetRoleBindingManager(),
				),
			}
		}).(*SFedRoleBindingClusterManager)
		GetFedRoleBindingManager().SetJointModelManager(FedRoleBindingClusterManager)
		RegisterFedJointClusterManager(GetFedRoleBindingManager(), FedRoleBindingClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedrolebindingcluster
// +onecloud:swagger-gen-model-plural=federatedrolebindingclusters
type SFedRoleBindingClusterManager struct {
	SFederatedNamespaceJointClusterManager
}

type SFedRoleBindingCluster struct {
	SFederatedNamespaceJointCluster
}

func (obj *SFedRoleBindingCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedRoleBindingCluster) GetFedRoleBinding() (*SFederatedRoleBinding, error) {
	fedObj, err := obj.GetFedResourceModel()
	if err != nil {
		return nil, err
	}
	return fedObj.(*SFederatedRoleBinding), nil
}

func (obj *SFedRoleBindingCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, base api.ClusterResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj, err := obj.GetFedRoleBinding()
	if err != nil {
		return nil, errors.Wrapf(err, "get federated rolebinding object")
	}
	nsBase, err := obj.SFederatedNamespaceJointCluster.GetResourceCreateInput(userCred, base)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster namespace base create input")
	}
	input := api.RoleBindingCreateInput{
		NamespaceResourceCreateInput: nsBase,
		Subjects:                     fedObj.Spec.Template.Subjects,
	}
	return input.JSON(input), nil
}
