package node

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// Node is a presentation layer view of Kubernetes nodes. This means it is node plus additional
// augmented data we can get from other sources.
type Node struct {
	api.ObjectMeta
	api.TypeMeta
	Ready              bool                   `json:"ready"`
	AllocatedResources NodeAllocatedResources `json:"allocatedResources"`
	// Addresses is a list of addresses reachable to the node. Queried from cloud provider, if available.
	Address []v1.NodeAddress `json:"addresses,omitempty"`
	// Set of ids/uuids to uniquely identify the node.
	NodeInfo v1.NodeSystemInfo `json:"nodeInfo"`
	// Taints
	Taints []v1.Taint `json:"taints,omitempty"`
	// Unschedulable controls node schedulability of new pods. By default node is schedulable.
	Unschedulable bool `json:"unschedulable"`
}

func toNode(node *v1.Node, pods []*v1.Pod, cluster api.ICluster) Node {
	allocatedResources, err := getNodeAllocatedResources(node, pods)
	if err != nil {
		log.Errorf("Couldn't get allocated resources of %s node: %s\n", node.Name, err)
	}

	return Node{
		ObjectMeta:         api.NewObjectMeta(node.ObjectMeta, cluster),
		TypeMeta:           api.NewTypeMeta(api.ResourceKindNode),
		Ready:              getNodeConditionStatus(node, v1.NodeReady),
		AllocatedResources: allocatedResources,
		NodeInfo:           node.Status.NodeInfo,
		Address:            node.Status.Addresses,
		Taints:             node.Spec.Taints,
		Unschedulable:      node.Spec.Unschedulable,
	}
}

func getNodeConditionStatus(node *v1.Node, conditionType v1.NodeConditionType) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}

// getContainerImages returns container image strings from the given node.
func getContainerImages(node v1.Node) []string {
	var containerImages []string
	for _, image := range node.Status.Images {
		for _, name := range image.Names {
			containerImages = append(containerImages, name)
		}
	}
	return containerImages
}

func getNodeConditions(pod v1.Node) []common.Condition {
	var conditions []common.Condition
	for _, condition := range pod.Status.Conditions {
		conditions = append(conditions, common.Condition{
			Type:               string(condition.Type),
			Status:             condition.Status,
			LastProbeTime:      condition.LastHeartbeatTime,
			LastTransitionTime: condition.LastTransitionTime,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	return conditions
}
