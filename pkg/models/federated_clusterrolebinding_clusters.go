package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	FedClusterRoleBindingClusterManager *SFedClusterRoleBindingClusterManager
)

func init() {
	db.InitManager(func() {
		FedClusterRoleBindingClusterManager = NewFederatedJointManager(func() db.IJointModelManager {
			return &SFedClusterRoleBindingClusterManager{
				SFederatedJointClusterManager: NewFederatedJointClusterManager(
					SFedClusterRoleBindingCluster{},
					"federatedclusterrolebindingclusters_tbl",
					"federatedclusterrolebindingcluster",
					"federatedclusterrolebindingclusters",
					GetFedClusterRoleBindingManager(),
					GetClusterRoleBindingManager(),
				),
			}
		}).(*SFedClusterRoleBindingClusterManager)
		GetFedClusterRoleBindingManager().SetJointModelManager(FedClusterRoleBindingClusterManager)
		RegisterFedJointClusterManager(GetFedClusterRoleBindingManager(), FedClusterRoleBindingClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedclusterrolebindingcluster
// +onecloud:swagger-gen-model-plural=federatedclusterrolebindingclusters
type SFedClusterRoleBindingClusterManager struct {
	SFederatedJointClusterManager
}

type SFedClusterRoleBindingCluster struct {
	SFederatedJointCluster
}

func (m *SFedClusterRoleBindingClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FedClusterRoleBindingClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFederatedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SFedClusterRoleBindingCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedClusterRoleBindingCluster) GetFedClusterRoleBinding() (*SFederatedClusterRoleBinding, error) {
	fObj, err := obj.GetFedResourceModel()
	if err != nil {
		return nil, errors.Wrap(err, "get federated clusterrolebinding")
	}
	return fObj.(*SFederatedClusterRoleBinding), nil
}

func (obj *SFedClusterRoleBindingCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, base api.ClusterResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj, err := obj.GetFedClusterRoleBinding()
	if err != nil {
		return nil, errors.Wrap(err, "get federated clusterrolebinding object")
	}
	input := api.ClusterRoleBindingCreateInput{
		ClusterResourceCreateInput: base,
		Subjects:                   fedObj.Spec.Template.Subjects,
		RoleRef:                    api.RoleRef(fedObj.Spec.Template.RoleRef),
	}
	return input.JSON(input), nil
}
