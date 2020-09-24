package api

import "yunion.io/x/onecloud/pkg/apis"

const (
	FederatedResourceStatusNotBind = "not_bind"
)

type FedJointClusterListInput struct {
	apis.JointResourceBaseListInput

	FederatedResourceId string `json:"federatedresource_id"`
	ClusterId           string `json:"cluster_id"`
	NamespaceId         string `json:"namespace_id"`
	ResourceId          string `json:"resource_id"`
}

type FedNamespaceJointClusterListInput struct {
	FedJointClusterListInput
	FederatedNamespaceId string `json:"federatednamespace_id"`
}
