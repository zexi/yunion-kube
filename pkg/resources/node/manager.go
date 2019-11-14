package node

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var NodeManager *SNodeManager

type SNodeManager struct {
	*resources.SClusterResourceManager
}

func init() {
	NodeManager = &SNodeManager{
		SClusterResourceManager: resources.NewClusterResourceManager("k8s_node", "k8s_nodes"),
	}
}

func (m *SNodeManager) AllowPerformAction(req *common.Request, id string) bool {
	return m.AllowUpdateItem(req, id)
}

func (m *SNodeManager) AllowPerformCordon(req *common.Request, id string) bool {
	return m.AllowPerformAction(req, id)
}

func (m *SNodeManager) PerformCordon(req *common.Request, id string) (interface{}, error) {
	return SetNodeScheduleToggle(req, id, true)
}

func (m *SNodeManager) AllowPerformUncordon(req *common.Request, id string) bool {
	return m.AllowPerformAction(req, id)
}

func (m *SNodeManager) PerformUncordon(req *common.Request, id string) (interface{}, error) {
	return SetNodeScheduleToggle(req, id, false)
}

func SetNodeScheduleToggle(req *common.Request, id string, unschedule bool) (*v1.Node, error) {
	cli := req.GetK8sAdminClient()
	indexer := req.GetIndexer()
	node, err := indexer.NodeLister().Get(id)
	if err != nil {
		return nil, err
	}
	nodeObj := node.DeepCopy()
	nodeObj.Spec.Unschedulable = unschedule
	return cli.CoreV1().Nodes().Update(nodeObj)
}
