package node

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
)

func (man *SNodeManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetNodeDetail(req.GetIndexer(), req.GetCluster(), id, dataselect.DefaultDataSelect())
}

// GetNodeDetail gets node details.
func GetNodeDetail(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	name string,
	dsQuery *dataselect.DataSelectQuery,
) (*api.NodeDetail, error) {
	log.Infof("Getting details of %s node", name)
	node, err := indexer.NodeLister().Get(name)
	if err != nil {
		return nil, err
	}

	pods, err := getNodePods(indexer, node)
	if err != nil {
		return nil, err
	}

	podList, err := GetNodePods(indexer, cluster, dsQuery, name)
	if err != nil {
		return nil, err
	}

	eventList, err := event.GetNodeEvents(indexer, cluster, dsQuery, node.UID)
	if err != nil {
		return nil, err
	}

	allocatedResources, err := getNodeAllocatedResources(node, pods)
	if err != nil {
		return nil, err
	}

	nodeDetails := toNodeDetail(*node, pods, podList, eventList, allocatedResources, cluster)
	return &nodeDetails, nil
}

func getNodeAllocatedResources(node *v1.Node, pods []*v1.Pod) (api.NodeAllocatedResources, error) {
	reqs, limits := map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}

	for _, pod := range pods {
		podReqs, podLimits, err := PodRequestsAndLimits(pod)
		if err != nil {
			return api.NodeAllocatedResources{}, err
		}
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = *podReqValue.Copy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = *podLimitValue.Copy()
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

	return api.NodeAllocatedResources{
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
				reqs[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = *quantity.Copy()
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
				reqs[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = *quantity.Copy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = *quantity.Copy()
			}
		}
	}
	return
}

// GetNodePods return pods list in given named node
func GetNodePods(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	dsQuery *dataselect.DataSelectQuery,
	name string,
) (*pod.PodList, error) {
	podList := pod.PodList{
		Pods: []api.Pod{},
	}

	node, err := indexer.NodeLister().Get(name)
	if err != nil {
		return &podList, err
	}

	pods, err := getNodePods(indexer, node)
	if err != nil {
		return &podList, err
	}

	return pod.ToPodListByIndexerV2(indexer, pods, "", dsQuery, labels.Everything(), cluster)
}

func getNodePods(client *client.CacheFactory, node *v1.Node) ([]*v1.Pod, error) {
	pods, err := client.PodLister().Pods(v1.NamespaceAll).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	rs := make([]*v1.Pod, 0)
	for _, p := range pods {
		if p.Spec.NodeName == node.Name && p.Status.Phase != v1.PodSucceeded && p.Status.Phase != v1.PodFailed {
			rs = append(rs, p)
		}
	}

	return rs, nil
}

func toNodeDetail(node v1.Node, rawPods []*v1.Pod, pods *pod.PodList, eventList *common.EventList,
	allocatedResources api.NodeAllocatedResources, cluster api.ICluster) api.NodeDetail {

	return api.NodeDetail{
		Node:               toNode(&node, rawPods, cluster),
		Phase:              node.Status.Phase,
		ProviderID:         node.Spec.ProviderID,
		PodCIDR:            node.Spec.PodCIDR,
		Conditions:         getNodeConditions(node),
		ContainerImages:    getContainerImages(node),
		PodList:            pods.GetPods(),
		EventList:          eventList.Events,
		AllocatedResources: allocatedResources,
		//Metrics:            metrics,
	}
}
