package node

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

func toNode(node *v1.Node, pods []*v1.Pod, cluster api.ICluster) api.Node {
	allocatedResources, err := getNodeAllocatedResources(node, pods)
	if err != nil {
		log.Errorf("Couldn't get allocated resources of %s node: %s\n", node.Name, err)
	}

	return api.Node{
		ObjectMeta:         api.NewObjectMeta(node.ObjectMeta, cluster),
		TypeMeta:           api.NewTypeMeta(node.TypeMeta),
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

func getNodeConditions(pod v1.Node) []api.Condition {
	var conditions []api.Condition
	for _, condition := range pod.Status.Conditions {
		conditions = append(conditions, api.Condition{
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
