package node

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

// NodeList contains a list of nodes in the cluster.
type NodeList struct {
	indexer *client.CacheFactory
	*common.BaseList
	Nodes []api.Node `json:"nodes"`
}

func (l *NodeList) GetResponseData() interface{} {
	return l.Nodes
}

func (man *SNodeManager) List(req *common.Request) (common.ListResource, error) {
	return GetNodeList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

// GetNodeListFromChannels returns a list of all Nodes in the cluster.
func GetNodeListFromChannels(indexer *client.CacheFactory, channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*NodeList, error) {

	nodes := <-channels.NodeList.List
	err := <-channels.NodeList.Error

	if err != nil {
		return nil, err
	}

	return toNodeList(indexer, nodes, dsQuery, cluster)
}

// GetNodeList returns a list of all Nodes in the cluster.
func GetNodeList(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*NodeList, error) {
	nodes, err := indexer.NodeLister().List(labels.Everything())
	if err != nil {
		return nil, err
	}

	return toNodeList(indexer, nodes, dsQuery, cluster)
}

func toNodeList(indexer *client.CacheFactory, nodes []*v1.Node, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*NodeList, error) {
	nodeList := &NodeList{
		BaseList: common.NewBaseList(cluster),
		indexer:  indexer,
		Nodes:    make([]api.Node, 0),
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
	node := obj.(*v1.Node)
	pods, err := getNodePods(l.indexer, node)
	if err != nil {
		log.Errorf("Couldn't get pods of %s node: %s\n", node.Name, err)
	}
	l.Nodes = append(l.Nodes, toNode(node, pods, l.GetCluster()))
}
