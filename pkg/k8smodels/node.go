package k8smodels

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	NodeManager *SNodeManager
	_           model.IK8SModel = &SNode{}
)

func init() {
	NodeManager = &SNodeManager{
		SK8SClusterResourceBaseManager: model.NewK8SClusterResourceBaseManager(
			&SNode{},
			"k8s_node",
			"k8s_nodes"),
	}
	NodeManager.SetVirtualObject(NodeManager)
}

type SNodeManager struct {
	model.SK8SClusterResourceBaseManager
}

type SNode struct {
	model.SK8SClusterResourceBase
}

func (m SNodeManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameNode,
		Object:       &v1.Node{},
	}
}

func (m SNodeManager) ValidateCreateData(
	ctx *model.RequestContext,
	query, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewUnsupportOperationError("kubernetes node not support create")
}

func (m SNodeManager) ListItemFilter(ctx *model.RequestContext, q model.IQuery, query *apis.ListInputNode) (model.IQuery, error) {
	q, err := m.SK8SClusterResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SClusterBase)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SNode) GetRawNode() *v1.Node {
	return obj.GetK8SObject().(*v1.Node)
}

func (obj *SNode) getNodeConditionStatus(node *v1.Node, conditionType v1.NodeConditionType) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}

// getContainerImages returns container image strings from the given node.
func (obj *SNode) getContainerImages(node v1.Node) []string {
	var containerImages []string
	for _, image := range node.Status.Images {
		for _, name := range image.Names {
			containerImages = append(containerImages, name)
		}
	}
	return containerImages
}

func (obj *SNode) getNodeConditions(node v1.Node) []*apis.Condition {
	var conditions []*apis.Condition
	for _, condition := range node.Status.Conditions {
		conditions = append(conditions, &apis.Condition{
			Type:               string(condition.Type),
			Status:             condition.Status,
			LastProbeTime:      condition.LastHeartbeatTime,
			LastTransitionTime: condition.LastTransitionTime,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	return SortConditions(conditions)
}

func (obj *SNode) getNodeAllocatedResources(node *v1.Node, pods []*v1.Pod) (apis.NodeAllocatedResources, error) {
	reqs, limits := map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}

	for _, pod := range pods {
		podReqs, podLimits, err := PodRequestsAndLimits(pod)
		if err != nil {
			return apis.NodeAllocatedResources{}, err
		}
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = podReqValue.DeepCopy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = podLimitValue.DeepCopy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}

	cpuRequests, cpuLimits, memoryRequests, memoryLimits := reqs[v1.ResourceCPU],
		limits[v1.ResourceCPU], reqs[v1.ResourceMemory], limits[v1.ResourceMemory]

	var cpuRequestsFraction, cpuLimitsFraction float64 = 0, 0
	if capacity := float64(node.Status.Capacity.Cpu().MilliValue()); capacity > 0 {
		cpuRequestsFraction = float64(cpuRequests.MilliValue()) / capacity * 100
		cpuLimitsFraction = float64(cpuLimits.MilliValue()) / capacity * 100
	}

	var memoryRequestsFraction, memoryLimitsFraction float64 = 0, 0
	if capacity := float64(node.Status.Capacity.Memory().MilliValue()); capacity > 0 {
		memoryRequestsFraction = float64(memoryRequests.MilliValue()) / capacity * 100
		memoryLimitsFraction = float64(memoryLimits.MilliValue()) / capacity * 100
	}

	var podFraction float64 = 0
	var podCapacity int64 = node.Status.Capacity.Pods().Value()
	if podCapacity > 0 {
		podFraction = float64(len(pods)) / float64(podCapacity) * 100
	}

	return apis.NodeAllocatedResources{
		CPURequests:            cpuRequests.MilliValue(),
		CPURequestsFraction:    cpuRequestsFraction,
		CPULimits:              cpuLimits.MilliValue(),
		CPULimitsFraction:      cpuLimitsFraction,
		CPUCapacity:            node.Status.Capacity.Cpu().MilliValue(),
		MemoryRequests:         memoryRequests.Value(),
		MemoryRequestsFraction: memoryRequestsFraction,
		MemoryLimits:           memoryLimits.Value(),
		MemoryLimitsFraction:   memoryLimitsFraction,
		MemoryCapacity:         node.Status.Capacity.Memory().Value(),
		AllocatedPods:          len(pods),
		PodCapacity:            podCapacity,
		PodFraction:            podFraction,
	}, nil
}

// PodRequestsAndLimits returns a dictionary of all defined resources summed up for all
// containers of the pod.
func PodRequestsAndLimits(pod *v1.Pod) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity, err error) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			value, ok := reqs[name]
			if !ok {
				reqs[name] = quantity.DeepCopy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = quantity.DeepCopy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = quantity.DeepCopy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = quantity.DeepCopy()
			}
		}
	}
	return
}

func (obj *SNode) GetRawPods() ([]*v1.Pod, error) {
	rNode := obj.GetRawNode()
	alllPods, err := PodManager.GetAllRawPods(obj.GetCluster())
	if err != nil {
		return nil, err
	}
	ret := make([]*v1.Pod, 0)
	for _, p := range alllPods {
		if p.Spec.NodeName == rNode.Name && p.Status.Phase != v1.PodSucceeded && p.Status.Phase != v1.PodFailed {
			ret = append(ret, p)
		}
	}
	return ret, nil
}

func (obj *SNode) GetAPIObject() (*apis.Node, error) {
	rNode := obj.GetRawNode()
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	allocatedResources, err := obj.getNodeAllocatedResources(rNode, pods)
	if err != nil {
		return nil, err
	}
	return &apis.Node{
		ObjectMeta:         obj.GetObjectMeta(),
		TypeMeta:           obj.GetTypeMeta(),
		Ready:              obj.getNodeConditionStatus(rNode, v1.NodeReady),
		AllocatedResources: allocatedResources,
		Address:            rNode.Status.Addresses,
		NodeInfo:           rNode.Status.NodeInfo,
		Taints:             rNode.Spec.Taints,
		Unschedulable:      rNode.Spec.Unschedulable,
	}, nil
}

func (obj *SNode) GetAPIDetailObject() (*apis.NodeDetail, error) {
	node, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	rNode := obj.GetRawNode()
	events, err := EventManager.GetEventsByObject(obj)
	if err != nil {
		return nil, err
	}
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	apiPods, err := PodManager.GetAPIPods(obj.GetCluster(), pods)
	if err != nil {
		return nil, err
	}
	return &apis.NodeDetail{
		Node:            *node,
		Phase:           rNode.Status.Phase,
		PodCIDR:         rNode.Spec.PodCIDR,
		ProviderID:      rNode.Spec.ProviderID,
		Conditions:      obj.getNodeConditions(*rNode),
		ContainerImages: obj.getContainerImages(*rNode),
		PodList:         apiPods,
		EventList:       events,
	}, nil
}

func (obj *SNode) PerformCordon(ctx *model.RequestContext, query, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, obj.SetNodeScheduleToggle(true)
}

func (obj *SNode) PerformUncordon(ctx *model.RequestContext, query, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, obj.SetNodeScheduleToggle(false)
}

func (obj *SNode) SetNodeScheduleToggle(unschedule bool) error {
	cli := obj.GetCluster().GetHandler()
	node := obj.GetRawNode()
	nodeObj := node.DeepCopy()
	nodeObj.Spec.Unschedulable = unschedule
	if _, err := cli.UpdateV2(apis.ResourceNameNode, "", nodeObj); err != nil {
		return err
	}
	return nil
}
