package models

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	FedNamespaceClusterManager *SFederatedNamespaceClusterManager
	_                          IFederatedJointClusterModel = new(SFederatedNamespaceCluster)
)

func init() {
	db.InitManager(func() {
		FedNamespaceClusterManager = NewFederatedJointManager(func() db.IJointModelManager {
			return &SFederatedNamespaceClusterManager{
				SFederatedJointClusterManager: NewFederatedJointClusterManager(
					SFederatedNamespaceCluster{},
					"federatednamespaceclusters_tbl",
					"federatednamespacecluster",
					"federatednamespaceclusters",
					GetFedNamespaceManager(),
					GetNamespaceManager(),
				),
			}
		}).(*SFederatedNamespaceClusterManager)
		GetFedNamespaceManager().SetJointModelManager(FedNamespaceClusterManager)
		RegisterFedJointClusterManager(GetFedNamespaceManager(), FedNamespaceClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatednamespacecluster
// +onecloud:swagger-gen-model-plural=federatednamespaceclusters
type SFederatedNamespaceClusterManager struct {
	SFederatedJointClusterManager
}

type SFederatedNamespaceCluster struct {
	SFederatedJointCluster
}

func (m *SFederatedNamespaceClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedNamespaceClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFederatedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SFederatedNamespaceCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFederatedNamespaceCluster) GetFedNamespace() (*SFederatedNamespace, error) {
	fObj, err := obj.GetFedResourceModel()
	if err != nil {
		return nil, errors.Wrap(err, "get federated namespace")
	}
	return fObj.(*SFederatedNamespace), nil
}

func (obj *SFederatedNamespaceCluster) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedNamespaceClusterDetails{
		FedJointClusterResourceDetails: obj.SFederatedJointCluster.GetDetails(base, isList).(api.FedJointClusterResourceDetails),
	}
	return out
}

func (obj *SFederatedNamespaceCluster) GetK8sResource() (runtime.Object, error) {
	fedNs, err := obj.GetFedNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "get federated namespace")
	}
	ns := &corev1.Namespace{
		ObjectMeta: fedNs.GetK8sObjectMeta(),
		Spec:       fedNs.Spec.Template.Spec,
	}
	return ns, nil
}

func (obj *SFederatedNamespaceCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, base api.ClusterResourceCreateInput) (jsonutils.JSONObject, error) {
	input := api.NamespaceCreateInputV2{
		ClusterResourceCreateInput: base,
	}
	return input.JSON(input), nil
}

func (obj *SFederatedNamespaceCluster) UpdateResource(resObj IClusterModel) error {
	return nil
}
