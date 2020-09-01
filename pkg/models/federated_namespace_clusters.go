package models

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	FederatedNamespaceClusterManager *SFederatedNamespaceClusterManager
	_                                IFederatedJointClusterModel = new(SFederatedNamespaceCluster)
)

func init() {
	db.InitManager(func() {
		FederatedNamespaceClusterManager = NewFederatedJointManager(func() db.IJointModelManager {
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
		GetFedNamespaceManager().SetJointModelManager(FederatedNamespaceClusterManager)
		RegisterFedJointClusterManager(GetFedNamespaceManager(), FederatedNamespaceClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatednamespacecluster
// +onecloud:swagger-gen-model-plural=federatednamespaceclusters
type SFederatedNamespaceClusterManager struct {
	SFederatedJointClusterManager
}

type SFederatedNamespaceCluster struct {
	SFederatedJointCluster

	FederatednamespaceId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (m *SFederatedNamespaceClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedNamespaceClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFederatedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FederatedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	if len(input.FederatednamespaceId) > 0 {
		fedNsObj, err := GetFedNamespaceManager().FetchByIdOrName(userCred, input.FederatednamespaceId)
		if err != nil {
			return nil, errors.Wrap(err, "Get federatednamespace object")
		}
		q = q.Equals("federatednamespace_id", fedNsObj.GetId())
	}
	return q, nil
}

func (obj *SFederatedNamespaceCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFederatedNamespaceCluster) GetFedNamespace() (*SFederatedNamespace, error) {
	return GetFedNamespaceManager().GetFedNamespace(obj.FederatednamespaceId)
}

func (obj *SFederatedNamespaceCluster) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedNamespaceClusterDetails{
		FederatedJointClusterResourceDetails: obj.SFederatedJointCluster.GetDetails(base, isList).(api.FederatedJointClusterResourceDetails),
	}
	if fedNs, err := obj.GetFedNamespace(); err != nil {
		log.Errorf("get federatednamespace %s object error: %v", obj.FederatednamespaceId, err)
	} else {
		out.Federatednamespace = fedNs.GetName()
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

func (obj *SFederatedNamespaceCluster) GetResourceCreateData(base api.ClusterResourceCreateInput) (jsonutils.JSONObject, error) {
	input := api.NamespaceCreateInputV2{
		ClusterResourceCreateInput: base,
	}
	return input.JSON(input), nil
}

func (obj *SFederatedNamespaceCluster) UpdateResource(resObj IClusterModel) error {
	return nil
}
