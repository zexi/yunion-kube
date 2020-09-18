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
	FedClusterRoleClusterManager *SFedClusterRoleClusterManager
	_                            IFedJointClusterModel = new(SFedClusterRoleCluster)
)

func init() {
	db.InitManager(func() {
		FedClusterRoleClusterManager = NewFedJointManager(func() db.IJointModelManager {
			return &SFedClusterRoleClusterManager{
				SFedJointClusterManager: NewFedJointClusterManager(
					SFedClusterRoleCluster{},
					"federatedclusterroleclusters_tbl",
					"federatedclusterrolecluster",
					"federatedclusterroleclusters",
					GetFedClusterRoleManager(),
					GetClusterRoleManager(),
				),
			}
		}).(*SFedClusterRoleClusterManager)
		GetFedClusterRoleManager().SetJointModelManager(FedClusterRoleClusterManager)
		RegisterFedJointClusterManager(GetFedClusterRoleManager(), FedClusterRoleClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedclusterrolecluster
// +onecloud:swagger-gen-model-plural=federatedclusterroleclusters
type SFedClusterRoleClusterManager struct {
	SFedJointClusterManager
}

type SFedClusterRoleCluster struct {
	SFedJointCluster
}

func (m *SFedClusterRoleClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedClusterRoleClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SFedClusterRoleCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedClusterRoleCluster) GetFedClusterRole() (*SFedClusterRole, error) {
	fedObj, err := GetFedResAPI().JointResAPI().FetchFedResourceModel(obj)
	if err != nil {
		return nil, err
	}
	return fedObj.(*SFedClusterRole), nil
}

func (obj *SFedClusterRoleCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, base api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj, err := obj.GetFedClusterRole()
	if err != nil {
		return nil, errors.Wrap(err, "get federated cluster role object")
	}
	input := api.ClusterRoleCreateInput{
		ClusterResourceCreateInput: base.ClusterResourceCreateInput,
		Rules:                      fedObj.Spec.Template.Rules,
	}
	return input.JSON(input), nil
}
