package models

import (
	"context"

	"k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	//"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	NodeManager *SNodeManager
)

func init() {
	NodeManager = &SNodeManager{
		SClusterResourceBaseManager: NewClusterResourceBaseManager(
			&SNode{},
			"nodes_tbl",
			"k8s_node",
			"k8s_nodes",
			api.ResourceNameNode,
			api.KindNameNode,
			new(v1.Node),
		),
	}
	NodeManager.SetVirtualObject(NodeManager)
	RegisterK8sModelManager(NodeManager)
}

type SNodeManager struct {
	SClusterResourceBaseManager
}

type SNode struct {
	SClusterResourceBase

	// v1.NodeSystemInfo
	NodeInfo jsonutils.JSONObject `list:"user"`
	// v1.NodeAddress
	Address jsonutils.JSONObject `list:"user"`

	// CpuCapacity is specified node CPU capacity in milicores
	CpuCapacity int64 `list:"user"`

	// MemoryCapacity is specified node memory capacity in bytes
	MemoryCapacity int64 `list:"user"`

	// PodCapacity is maximum number of pods that can be allocated on given node
	PodCapacity int64 `list:"user"`
}

func (m *SNodeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.NodeCreateInput) (*api.NodeCreateInput, error) {
	return nil, httperrors.NewBadRequestError("Not support node create")
}

func (m *SNodeManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NodeListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.ClusterResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SNodeManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.NodeDetailV2 {
	rows := make([]api.NodeDetailV2, len(objs))
	cRows := m.SClusterResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range cRows {
		detail := api.NodeDetailV2{
			ClusterResourceDetail: cRows[i],
		}
		rows[i] = detail
	}
	return rows
}

func (node *SNode) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (*jsonutils.JSONDict, error) {
	return nil, nil
}

func (node *SNode) moreExtraInfo(detail *api.NodeDetailV2) *api.NodeDetailV2 {
	return detail
}

// NewFromRemoteObject create local db SNode model by remote k8s node object
func (m *SNodeManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	return m.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
}

func (node *SNode) getNodeConditionStatus(k8sNode *v1.Node, conditionType v1.NodeConditionType) v1.ConditionStatus {
	for _, condition := range k8sNode.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return v1.ConditionUnknown
}

func (node *SNode) getStatusFromRemote(k8sNode *v1.Node) string {
	readyCondStatus := node.getNodeConditionStatus(k8sNode, v1.NodeReady)
	if readyCondStatus == v1.ConditionTrue {
		return api.NodeStatusReady
	}
	return api.NodeStatusNotReady
}

// UpdateFromRemoteObject update local db SNode model by remote k8s node object
func (node *SNode) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := node.SClusterResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	k8sNode := extObj.(*v1.Node)
	node.Address = jsonutils.Marshal(k8sNode.Status.Addresses)
	node.NodeInfo = jsonutils.Marshal(k8sNode.Status.NodeInfo)
	node.Status = node.getStatusFromRemote(k8sNode)
	node.updateCapacity(k8sNode)
	return nil
}

func (node *SNode) updateCapacity(k8sNode *v1.Node) {
	capacity := k8sNode.Status.Capacity
	// cpu status
	node.CpuCapacity = capacity.Cpu().MilliValue()
	// memory status
	node.MemoryCapacity = capacity.Memory().MilliValue()
	// pod status
	node.PodCapacity = capacity.Pods().Value()
}

func (m *SNodeManager) GetNodesByClusters(clusterIds []string) ([]SNode, error) {
	nodes := make([]SNode, 0)
	if err := GetResourcesByClusters(m, clusterIds, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (m *SNodeManager) getNodePods(node *SNode, pods []SPod) []SPod {
	ret := make([]SPod, 0)
	for _, p := range pods {
		if p.NodeId == node.GetId() {
			ret = append(ret, p)
		}
	}
	return ret
}

func (m *SNodeManager) Usage(clusters []sClusterUsage) (*api.NodeUsage, error) {
	clusterIds := make([]string, len(clusters))
	for i := range clusters {
		clusterIds[i] = clusters[i].Id
	}
	pods, err := PodManager.GetPodsByClusters(clusterIds)
	if err != nil {
		return nil, err
	}
	nodes, err := m.GetNodesByClusters(clusterIds)
	if err != nil {
		return nil, err
	}
	fillUsage := func(pods []SPod, nu *api.NodeUsage) *api.NodeUsage {
		for _, p := range pods {
			nu.Memory.Request += p.MemoryRequests
			nu.Memory.Limit += p.MemoryLimits
			nu.Cpu.Request += p.CpuRequests
			nu.Cpu.Limit += p.CpuLimits
		}
		return nu
	}
	eachUsages := make([]*api.NodeUsage, 0)
	for _, node := range nodes {
		nPods := m.getNodePods(&node, pods)
		nu := api.NewNodeUsage()
		nu.Count = 1
		if node.Status == api.NodeStatusReady {
			nu.ReadyCount = 1
		} else {
			nu.NotReadyCount = 1
		}
		nu.Memory = &api.MemoryUsage{
			Capacity: node.MemoryCapacity,
		}
		nu.Cpu = &api.CpuUsage{
			Capacity: node.CpuCapacity,
		}
		nu.Pod = &api.PodUsage{
			Count:    int64(len(nPods)),
			Capacity: node.PodCapacity,
		}
		eachUsages = append(eachUsages, fillUsage(nPods, nu))
	}

	totalUsage := api.NewNodeUsage()
	for _, each := range eachUsages {
		totalUsage.Add(each)
	}
	//log.Errorf("==result usage: %v", jsonutils.Marshal(totalUsage))
	return totalUsage, nil
}
