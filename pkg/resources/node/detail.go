package node

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8sClient "k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// NodeAllocatedResources describes node allocated resources.
type NodeAllocatedResources struct {
	// CPURequests is number of allocated milicores.
	CPURequests int64 `json:"cpuRequests"`

	// CPURequestsFraction is a fraction of CPU, that is allocated.
	CPURequestsFraction float64 `json:"cpuRequestsFraction"`

	// CPULimits is defined CPU limit.
	CPULimits int64 `json:"cpuLimits"`

	// CPULimitsFraction is a fraction of defined CPU limit, can be over 100%, i.e.
	// overcommitted.
	CPULimitsFraction float64 `json:"cpuLimitsFraction"`

	// CPUCapacity is specified node CPU capacity in milicores.
	CPUCapacity int64 `json:"cpuCapacity"`

	// MemoryRequests is a fraction of memory, that is allocated.
	MemoryRequests int64 `json:"memoryRequests"`

	// MemoryRequestsFraction is a fraction of memory, that is allocated.
	MemoryRequestsFraction float64 `json:"memoryRequestsFraction"`

	// MemoryLimits is defined memory limit.
	MemoryLimits int64 `json:"memoryLimits"`

	// MemoryLimitsFraction is a fraction of defined memory limit, can be over 100%, i.e.
	// overcommitted.
	MemoryLimitsFraction float64 `json:"memoryLimitsFraction"`

	// MemoryCapacity is specified node memory capacity in bytes.
	MemoryCapacity int64 `json:"memoryCapacity"`

	// AllocatedPods in number of currently allocated pods on the node.
	AllocatedPods int `json:"allocatedPods"`

	// PodCapacity is maximum number of pods, that can be allocated on the node.
	PodCapacity int64 `json:"podCapacity"`

	// PodFraction is a fraction of pods, that can be allocated on given node.
	PodFraction float64 `json:"podFraction"`
}

// NodeDetail is a presentation layer view of Kubernetes Node resource. This means it is Node plus
// additional augmented data we can get from other sources.
type NodeDetail struct {
	api.ObjectMeta
	api.TypeMeta

	// NodePhase is the current lifecycle phase of the node.
	Phase v1.NodePhase `json:"status"`

	// Resources allocated by node.
	AllocatedResources NodeAllocatedResources `json:"allocatedResources"`

	// PodCIDR represents the pod IP range assigned to the node.
	PodCIDR string `json:"podCIDR"`

	// ID of the node assigned by the cloud provider.
	ProviderID string `json:"providerID"`

	// Unschedulable controls node schedulability of new pods. By default node is schedulable.
	Unschedulable bool `json:"unschedulable"`

	// Set of ids/uuids to uniquely identify the node.
	NodeInfo v1.NodeSystemInfo `json:"nodeInfo"`

	// Conditions is an array of current node conditions.
	Conditions []common.Condition `json:"conditions"`

	// Container images of the node.
	ContainerImages []string `json:"containerImages"`

	// PodList contains information about pods belonging to this node.
	PodList pod.PodList `json:"podList"`

	// Events is list of events associated to the node.
	EventList common.EventList `json:"eventList"`

	// Metrics collected for this resource
	//Metrics []metricapi.Metric `json:"metrics"`

	// Taints
	Taints []v1.Taint `json:"taints,omitempty"`

	// Addresses is a list of addresses reachable to the node. Queried from cloud provider, if available.
	Addresses []v1.NodeAddress `json:"addresses,omitempty"`
}

func (man *SNodeManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetNodeDetail(req.GetK8sClient(), req.GetCluster(), id, dataselect.DefaultDataSelect())
}

// GetNodeDetail gets node details.
func GetNodeDetail(
	client k8sClient.Interface,
	cluster api.ICluster,
	name string,
	dsQuery *dataselect.DataSelectQuery,
) (*NodeDetail, error) {
	log.Infof("Getting details of %s node", name)
	node, err := client.CoreV1().Nodes().Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	pods, err := getNodePods(client, *node)
	if err != nil {
		return nil, err
	}

	podList, err := GetNodePods(client, cluster, dsQuery, name)
	if err != nil {
		return nil, err
	}

	eventList, err := event.GetNodeEvents(client, dsQuery, node.Name)
	if err != nil {
		return nil, err
	}

	allocatedResources, err := getNodeAllocatedResources(*node, pods)
	if err != nil {
		return nil, err
	}

	nodeDetails := toNodeDetail(*node, podList, &eventList, allocatedResources)
	return &nodeDetails, nil
}

func getNodeAllocatedResources(node v1.Node, podList *v1.PodList) (NodeAllocatedResources, error) {
	reqs, limits := map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}

	for _, pod := range podList.Items {
		podReqs, podLimits, err := PodRequestsAndLimits(&pod)
		if err != nil {
			return NodeAllocatedResources{}, err
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
		podFraction = float64(len(podList.Items)) / float64(podCapacity) * 100
	}

	return NodeAllocatedResources{
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
		AllocatedPods:          len(podList.Items),
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
	client k8sClient.Interface,
	cluster api.ICluster,
	dsQuery *dataselect.DataSelectQuery,
	name string,
) (*pod.PodList, error) {
	podList := pod.PodList{
		Pods: []pod.Pod{},
	}

	node, err := client.CoreV1().Nodes().Get(name, metaV1.GetOptions{})
	if err != nil {
		return &podList, err
	}

	pods, err := getNodePods(client, *node)
	if err != nil {
		return &podList, err
	}

	events, err := event.GetPodsEvents(client, v1.NamespaceAll, pods.Items)
	if err != nil {
		return &podList, err
	}

	return pod.ToPodList(pods.Items, events, dsQuery, cluster)
}

func getNodePods(client k8sClient.Interface, node v1.Node) (*v1.PodList, error) {
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node.Name +
		",status.phase!=" + string(v1.PodSucceeded) +
		",status.phase!=" + string(v1.PodFailed))

	if err != nil {
		return nil, err
	}

	return client.CoreV1().Pods(v1.NamespaceAll).List(metaV1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
}

func toNodeDetail(node v1.Node, pods *pod.PodList, eventList *common.EventList,
	allocatedResources NodeAllocatedResources) NodeDetail {

	return NodeDetail{
		ObjectMeta:         api.NewObjectMeta(node.ObjectMeta),
		TypeMeta:           api.NewTypeMeta(api.ResourceKindNode),
		Phase:              node.Status.Phase,
		ProviderID:         node.Spec.ProviderID,
		PodCIDR:            node.Spec.PodCIDR,
		Unschedulable:      node.Spec.Unschedulable,
		NodeInfo:           node.Status.NodeInfo,
		Conditions:         getNodeConditions(node),
		ContainerImages:    getContainerImages(node),
		PodList:            *pods,
		EventList:          *eventList,
		AllocatedResources: allocatedResources,
		//Metrics:            metrics,
		Taints:    node.Spec.Taints,
		Addresses: node.Status.Addresses,
	}
}
