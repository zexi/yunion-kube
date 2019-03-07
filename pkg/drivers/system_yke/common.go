package system_yke

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/drivers/yunion_host"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

func GetV1Roles(role string) []string {
	v1Roles := []string{}
	if role == types.RoleTypeControlplane {
		v1Roles = append(v1Roles, "etcd", "controlplane")
	}
	if role == types.RoleTypeNode {
		v1Roles = append(v1Roles, "worker")
	}
	return v1Roles
}

func GetAddNodeData(cluster *clusters.SCluster, m *types.CreateMachineData) (*apis.NodeAddOption, error) {
	v1Cluster, err := yunion_host.GetV1Cluster(cluster)
	if err != nil {
		return nil, err
	}

	ret := &apis.NodeAddOption{
		Cluster: v1Cluster.GetId(),
		Roles:   GetV1Roles(m.Role),
		Name:    m.Name,
		Host:    m.ResourceId,
		DockerdConfig: apis.DockerdConfig{
			LiveRestore: true,
			Graph:       models.DEFAULT_DOCKER_GRAPH_DIR,
		},
	}
	return ret, nil
}

func GetAddNodesData(cluster *clusters.SCluster, ms []*types.CreateMachineData) ([]*apis.NodeAddOption, error) {
	nodes := make([]*apis.NodeAddOption, 0)
	for _, m := range ms {
		addOpt, err := GetAddNodeData(cluster, m)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, addOpt)
	}
	return nodes, nil
}

func GetNodeByMachine(m manager.IMachine) (*models.SNode, error) {
	return models.NodeManager.FetchNodeByHostId(m.GetResourceId())
}

func GetNodesByMachines(ms []manager.IMachine) ([]*models.SNode, error) {
	ns := make([]*models.SNode, 0)
	for _, m := range ms {
		n, err := GetNodeByMachine(m)
		if err != nil {
			return nil, err
		}
		ns = append(ns, n)
	}
	return ns, nil
}

func GetClusterAddNodesData(cluster *clusters.SCluster, ms []*types.CreateMachineData) (*jsonutils.JSONDict, error) {
	nodes, err := GetAddNodesData(cluster, ms)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.Marshal(nodes), "nodes")
	ret.Add(jsonutils.JSONTrue, "auto_deploy")
	return ret, nil
}
