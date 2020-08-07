package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/onecloud/pkg/apis"
)

type FederatedResourceCreateInput struct {
	K8sResourceCreateInput
}

type FederatedResourceJointClusterInput struct {
	apis.Meta
	ClusterId string `json:"cluster_id"`
}

type FederatedNamespaceResourceCreateInput struct {
	FederatedResourceCreateInput
	FederatednamespaceId string `json:"federatednamespace_id"`
	Federatednamespace   string
}

func (input FederatedNamespaceResourceCreateInput) ToObjectMeta() metav1.ObjectMeta {
	objMeta := input.FederatedResourceCreateInput.ToObjectMeta()
	objMeta.Namespace = input.Federatednamespace
	return objMeta
}

type FederatedResourceDetails struct {
	apis.StatusDomainLevelResourceDetails
	Placement FederatedPlacement `json:"placement"`
}

type FederatedPlacement struct {
	Clusters []FederatedJointCluster `json:"clusters"`
}

type FederatedJointCluster struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type FederatedJointClusterResourceDetails struct {
	apis.JointResourceBaseDetails
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	Resource  string `json:"resource"`
}
