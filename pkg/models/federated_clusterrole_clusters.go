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
	FedClusterRoleClusterManager *SFederatedClusterRoleClusterManager
	_                            IFederatedJointClusterModel = new(SFederatedClusterRoleCluster)
)

func init() {
	db.InitManager(func() {
		FedClusterRoleClusterManager = NewFederatedJointManager(func() db.IJointModelManager {
			return &SFederatedClusterRoleClusterManager{
				SFederatedJointClusterManager: NewFederatedJointClusterManager(
					SFederatedClusterRoleCluster{},
					"federatedclusterroleclusters_tbl",
					"federatedclusterrolecluster",
					"federatedclusterroleclusters",
					GetFedClusterRoleManager(),
					GetClusterRoleManager(),
				),
			}
		}).(*SFederatedClusterRoleClusterManager)
		GetFedClusterRoleManager().SetJointModelManager(FedClusterRoleClusterManager)
		RegisterFedJointClusterManager(GetFedClusterRoleManager(), FedClusterRoleClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedclusterrolecluster
// +onecloud:swagger-gen-model-plural=federatedclusterroleclusters
type SFederatedClusterRoleClusterManager struct {
	SFederatedJointClusterManager
}

type SFederatedClusterRoleCluster struct {
	SFederatedJointCluster

	FederatedclusterroleId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (m *SFederatedClusterRoleClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedClusterRoleClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFederatedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SFederatedClusterRoleCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFederatedClusterRoleCluster) GetFedClusterRole() (*SFederatedClusterRole, error) {
	return GetFedClusterRoleManager().GetFedClusterRole(obj.FederatedclusterroleId)
}

func (obj *SFederatedClusterRoleCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, base api.ClusterResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj, err := obj.GetFedClusterRole()
	if err != nil {
		return nil, errors.Wrap(err, "get federated cluster role object")
	}
	input := api.ClusterRoleCreateInput{
		ClusterResourceCreateInput: base,
		Rules:                      fedObj.Spec.Template.Rules,
	}
	return input.JSON(input), nil
}
