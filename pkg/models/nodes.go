package models

import (
	"context"
	"errors"
	"fmt"

	yketypes "yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/httperrors"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/mcclient"
	"yunion.io/yunioncloud/pkg/sqlchemy"
	"yunion.io/yunioncloud/pkg/util/sets"

	"yunion.io/yunion-kube/pkg/types/apis"
)

var NodeManager *SNodeManager

var (
	NodeNotFoundError = errors.New("Node not found")
)

func init() {
	NodeManager = &SNodeManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SNode{}, "nodes_tbl", "node", "nodes"),
	}
}

type SNodeManager struct {
	db.SVirtualResourceBaseManager
}

func (m *SNodeManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return m.SVirtualResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func validateRoles(data jsonutils.JSONObject) (etcd, ctrl, worker bool, err error) {
	validRoles := sets.NewString("etcd", "controlplane", "worker")
	roles, err := data.GetArray("roles")
	if err != nil {
		return
	}
	if len(roles) == 0 {
		err = fmt.Errorf("Roles must provided")
		return
	}
	for _, reqRole := range roles {
		role := reqRole.String()
		if !validRoles.Has(role) {
			err = fmt.Errorf("Invalid role %q", role)
			break
		}
		switch role {
		case "etcd":
			etcd = true
		case "controlplane":
			ctrl = true
		case "worker":
			worker = true
		}
	}
	if !(etcd || ctrl || worker) {
		err = fmt.Errorf("Invalid roles: %#v", roles)
	}
	return
}

func (m *SNodeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	clusterIdent, _ := data.GetString("cluster")
	if clusterIdent == "" {
		return nil, httperrors.NewInputParameterError("Cluster must specified")
	}
	cluster := ClusterManager.FetchCluster(clusterIdent)
	if cluster == nil {
		return nil, httperrors.NewInputParameterError("Cluster %q not found", clusterIdent)
	}

	isEtcd, isCtrl, isWorker, err := validateRoles(data)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Cluster role: %v", err)
	}
	toBool := func(v bool) jsonutils.JSONObject {
		if v {
			return jsonutils.JSONTrue
		}
		return jsonutils.JSONFalse
	}
	data.Add(toBool(isEtcd), "etcd")
	data.Add(toBool(isCtrl), "controlplane")
	data.Add(toBool(isWorker), "worker")

	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (m *SNodeManager) FilterByOwner(q *sqlchemy.SQuery, ownerProjId string) *sqlchemy.SQuery {
	if len(ownerProjId) > 0 {
		q = q.Equals("tenant_id", ownerProjId)
	}
	return q
}

func (m *SNodeManager) FetchNode(ident string) *SNode {
	node, err := m.FetchByIdOrName("", ident)
	if err != nil {
		log.Errorf("Fetch node %q fail: %v", ident, err)
		return nil
	}
	return node.(*SNode)
}

func (m *SNodeManager) FetchClusterNode(cluster, ident string) *SNode {
	// TODO: impl this
	return m.FetchNode(ident)
}

func (m *SNodeManager) GetNode(cluster, ident string) (*apis.Node, error) {
	node := m.FetchNode(ident)
	if node == nil {
		return nil, NodeNotFoundError
	}
	return node.Node()
}

func (m *SNodeManager) ListByCluster(clusterId string) ([]*SNode, error) {
	nodes := m.Query().SubQuery()
	q := nodes.Query().Filter(sqlchemy.Equals(nodes.Field("cluster_id"), clusterId))
	objs := []SNode{}
	err := q.All(&objs)
	if err != nil {
		return nil, err
	}
	return ConvertPtrNodes(objs), nil
}

func ConvertPtrNodes(objs []SNode) []*SNode {
	ret := make([]*SNode, 0)
	for _, obj := range objs {
		ret = append(ret, &obj)
	}
	return ret
}

func mergePendingNodes(nodes, pendingNodes []*SNode) []*SNode {
	isIn := func(pnode *SNode, nodes []*SNode) (int, bool) {
		for idx, node := range nodes {
			log.Debugf("====pnode id: %v, node id: %v", pnode.Id, node.Id)
			if node.Id == pnode.Id {
				return idx, true
			}
		}
		return 0, false
	}

	for _, pnode := range pendingNodes {
		if idx, ok := isIn(pnode, nodes); ok {
			nodes[idx] = pnode
		} else {
			nodes = append(nodes, pnode)
		}
	}

	return nodes
}

type SNode struct {
	db.SVirtualResourceBase

	ClusterId string `nullable:"false" create:"required" list:"user"`

	Address           string `nullable:"true" list:"user"`
	Etcd              bool   `nullable:"true" default:"false" list:"user"`
	ControlPlane      bool   `nullable:"true" default:"false" list:"user"`
	Worker            bool   `nullable:"true" default:"false" list:"user"`
	RequestedHostname string `nullable:"true" list:"user"`
	HostnameOverride  string `nullable:"true" list:"user"`

	Labels     jsonutils.JSONObject `nullable:true`
	DockerInfo jsonutils.JSONObject `nullable:"true"`
}

func (n *SNode) Register(data *apis.Node) (*SNode, error) {
	if n.ClusterId != data.ClusterId {
		return nil, fmt.Errorf("ClusterId %q and %q not match", n.ClusterId, data.ClusterId)
	}

	if data.Address == "" {
		return nil, fmt.Errorf("Address must provided")
	}

	_, err := n.GetModelManager().TableSpec().Update(n, func() error {
		n.Address = data.Address
		if data.DockerInfo != nil {
			dInfo := jsonutils.Marshal(data.DockerInfo)
			n.DockerInfo = dInfo
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	f := ClusterManager.UpdateCluster
	if n.Status == "init" {
		f = ClusterManager.AddClusterNodes
	}

	err = f(n.ClusterId, n)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func (n *SNode) Node() (*apis.Node, error) {
	return &apis.Node{
		Name:         n.Name,
		Etcd:         n.Etcd,
		ControlPlane: n.ControlPlane,
		Worker:       n.Worker,
		NodeConfig:   n.GetNodeConfig(),
	}, nil
}

func (n *SNode) GetRoles() []string {
	roles := []string{}
	if n.Etcd {
		roles = append(roles, "etcd")
	}
	if n.ControlPlane {
		roles = append(roles, "controlplane")
	}
	if n.Worker {
		roles = append(roles, "worker")
	}
	return roles
}

func (n *SNode) GetLabels() (map[string]string, error) {
	labels := make(map[string]string)
	err := n.Labels.Unmarshal(labels)
	return labels, err
}

func (n *SNode) GetNodeConfig() *yketypes.ConfigNode {
	node := &yketypes.ConfigNode{
		NodeName:         n.Name,
		Address:          n.Address,
		Port:             "22",
		User:             "root",
		Role:             n.GetRoles(),
		HostnameOverride: n.HostnameOverride,
		DockerSocket:     "/var/run/docker.sock",
	}
	labels, err := n.GetLabels()
	if err != nil {
		log.Errorf("Get labels error: %v", err)
	} else {
		node.Labels = labels
	}
	return node
}

func (n *SNode) GetCluster() (*SCluster, error) {
	cluster := ClusterManager.FetchCluster(n.ClusterId)
	if cluster == nil {
		return nil, ClusterNotFoundError
	}
	return cluster, nil
}

func (n *SNode) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return n.IsOwner(userCred)
}

func (n *SNode) ValidateDeleteCondition(ctx context.Context) error {
	// TODO: validate cluster status, only can delete when cluster ready
	cluster, err := n.GetCluster()
	if err != nil {
		return err
	}
	if sets.NewString(CLUSTER_RUNNING, CLUSTER_UPDATING).Has(cluster.Status) {
		return fmt.Errorf("Can't delete node when cluster %q status is %q", cluster.Name, cluster.Status)
	}
	return nil
}

func (n *SNode) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	cluster, err := n.GetCluster()
	if err != nil {
		return err
	}
	config, _, err := ClusterManager.getConfig(false, cluster)
	if err != nil {
		return err
	}
	config = RemoveYKEConfigNode(config, n)
	return cluster.SetYKEConfig(config)
}

func RemoveYKEConfigNode(config *yketypes.KubernetesEngineConfig, rNode *SNode) *yketypes.KubernetesEngineConfig {
	ykeNodes := config.Nodes
	for i, n := range ykeNodes {
		if n.NodeName == rNode.Name {
			ykeNodes = append(ykeNodes[:i], ykeNodes[i+1:]...)
		}
	}
	config.Nodes = ykeNodes
	return config
}
