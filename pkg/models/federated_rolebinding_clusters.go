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
	_                            IFedJointClusterModel = new(SFedRoleBindingCluster)
)

func init() {
	db.InitManager(func() {
		FedRoleBindingClusterManager = NewFedJointManager(func() db.IJointModelManager {
			return &SFedRoleBindingClusterManager{
				SFedNamespaceJointClusterManager: NewFedNamespaceJointClusterManager(
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
	SFedNamespaceJointClusterManager
}

type SFedRoleBindingCluster struct {
	SFederatedNamespaceJointCluster
}

func (obj *SFedRoleBindingCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedRoleBindingCluster) GetFedRoleBinding() (*SFedRoleBinding, error) {
	fedObj, err := GetFedDBAPI().JointDBAPI().FetchFedResourceModel(obj)
	if err != nil {
		return nil, err
	}
	return fedObj.(*SFedRoleBinding), nil
}

func (obj *SFedRoleBindingCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, base api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj, err := obj.GetFedRoleBinding()
	if err != nil {
		return nil, errors.Wrapf(err, "get federated rolebinding object")
	}
	input := api.RoleBindingCreateInput{
		NamespaceResourceCreateInput: base,
		Subjects:                     fedObj.Spec.Template.Subjects,
	}
	return input.JSON(input), nil
}
