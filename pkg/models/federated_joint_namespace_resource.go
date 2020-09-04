package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

type SFederatedNamespaceJointClusterManager struct {
	SFederatedJointClusterManager
}

type SFederatedNamespaceJointCluster struct {
	SFederatedJointCluster
}

func NewFedNamespaceJointClusterManager(
	dt interface{}, tableName string,
	keyword string, keywordPlural string,
	master IFederatedModelManager,
	resourceMan IClusterModelManager,
) SFederatedNamespaceJointClusterManager {
	return SFederatedNamespaceJointClusterManager{
		SFederatedJointClusterManager: NewFederatedJointClusterManager(dt, tableName, keyword, keywordPlural, master, resourceMan),
	}
}

func (m *SFederatedNamespaceJointClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FedNamespaceJointClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFederatedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	if input.FederatedNamespaceId != "" {
		fedNsObj, err := GetFedNamespaceManager().FetchByIdOrName(userCred, input.FederatedNamespaceId)
		if err != nil {
			return nil, errors.Wrap(err, "Get federatednamespace")
		}
		masterSq := m.GetFedManager().Query("id").Equals("federatednamespace_id", fedNsObj.GetId()).SubQuery()
		q = q.In("federatedresource_id", masterSq)
	}
	return q, nil
}

func (obj *SFederatedNamespaceJointCluster) GetFedNamespace() (*SFederatedNamespace, error) {
	fedObj, err := obj.GetFedResourceModel()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get fed %s obj", obj.GetFedResourceManager().Keyword())
	}
	return fedObj.(IFederatedNamespaceModel).GetFedNamespace()
}

func (obj *SFederatedNamespaceJointCluster) GetClusterNamespace(userCred mcclient.TokenCredential, clusterId string, namespace string) (*SNamespace, error) {
	nsObj, err := GetNamespaceManager().GetByName(userCred, clusterId, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s namespace %s", clusterId, namespace)
	}
	return nsObj.(*SNamespace), nil
}

func (obj *SFederatedNamespaceJointCluster) GetResourceCreateInput(userCred mcclient.TokenCredential, base api.ClusterResourceCreateInput) (api.NamespaceResourceCreateInput, error) {
	fedNs, err := obj.GetFedNamespace()
	if err != nil {
		return api.NamespaceResourceCreateInput{}, errors.Wrap(err, "Get federated namespace")
	}
	nsObj, err := obj.GetClusterNamespace(userCred, base.ClusterId, fedNs.GetName())
	if err != nil {
		return api.NamespaceResourceCreateInput{}, errors.Wrap(err, "Get cluster namespace")
	}

	return api.NamespaceResourceCreateInput{
		ClusterResourceCreateInput: base,
		NamespaceId:                nsObj.GetId(),
	}, nil
}

/*
 * func (obj *SFederatedNamespaceJointCluster) GetDetails(base interface{}, isList bool) interface{} {
 *     out := api.FedNamespaceJointClusterResourceDetails{
 *         FedJointClusterResourceDetails: base.(api.FedJointClusterResourceDetails),
 *     }
 * }
 */
