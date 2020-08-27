package api

import "yunion.io/x/onecloud/pkg/apis"

type FederatedJointClusterListInput struct {
	apis.JointResourceBaseListInput

	ClusterId   string `json:"cluster_id"`
	NamespaceId string `json:"namespace_id"`
	ResourceId  string `json:"resource_id"`
}
