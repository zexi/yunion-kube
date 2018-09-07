package node

import (
	"k8s.io/api/core/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// NodeList contains a list of nodes in the cluster.
type NodeList struct {
	client client.Interface
	*dataselect.ListMeta
	Nodes []Node `json:"nodes"`
}

func (l *NodeList) GetResponseData() interface{} {
	return l.Nodes
}

func (man *SNodeManager) AllowListItems(req *common.Request) bool {
	return req.UserCred.IsSystemAdmin()
}

func (man *SNodeManager) List(req *common.Request) (common.ListResource, error) {
	return GetNodeList(req.GetK8sClient(), req.ToQuery())
}

// Node is a presentation layer view of Kubernetes nodes. This means it is node plus additional
// augmented data we can get from other sources.
type Node struct {
	api.ObjectMeta
	api.TypeMeta
	Ready              v1.ConditionStatus     `json:"ready"`
	AllocatedResources NodeAllocatedResources `json:"allocatedResources"`
}

// GetNodeListFromChannels returns a list of all Nodes in the cluster.
func GetNodeListFromChannels(client client.Interface, channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*NodeList, error) {

	nodes := <-channels.NodeList.List
	err := <-channels.NodeList.Error

	if err != nil {
		return nil, err
	}

	return toNodeList(client, nodes.Items, dsQuery)
}

// GetNodeList returns a list of all Nodes in the cluster.
func GetNodeList(client client.Interface, dsQuery *dataselect.DataSelectQuery) (*NodeList, error) {
	nodes, err := client.CoreV1().Nodes().List(api.ListEverything)
	if err != nil {
		return nil, err
	}

	return toNodeList(client, nodes.Items, dsQuery)
}

func toNodeList(client client.Interface, nodes []v1.Node, dsQuery *dataselect.DataSelectQuery) (*NodeList, error) {
	nodeList := &NodeList{
		client:   client,
		Nodes:    make([]Node, 0),
		ListMeta: dataselect.NewListMeta(),
	}

	err := dataselect.ToResourceList(
		nodeList,
		nodes,
		dataselect.NewNamespaceDataCell,
		dsQuery,
	)

	return nodeList, err

}
func (l *NodeList) Append(obj interface{}) {
	node := obj.(v1.Node)
	pods, err := getNodePods(l.client, node)
	if err != nil {
		log.Errorf("Couldn't get pods of %s node: %s\n", node.Name, err)
	}
	l.Nodes = append(l.Nodes, toNode(node, pods))
}

func toNode(node v1.Node, pods *v1.PodList) Node {
	allocatedResources, err := getNodeAllocatedResources(node, pods)
	if err != nil {
		log.Errorf("Couldn't get allocated resources of %s node: %s\n", node.Name, err)
	}

	return Node{
		ObjectMeta:         api.NewObjectMeta(node.ObjectMeta),
		TypeMeta:           api.NewTypeMeta(api.ResourceKindNode),
		Ready:              getNodeConditionStatus(node, v1.NodeReady),
		AllocatedResources: allocatedResources,
	}
}

func getNodeConditionStatus(node v1.Node, conditionType v1.NodeConditionType) v1.ConditionStatus {
	for _, condition := range node.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return v1.ConditionUnknown
}
