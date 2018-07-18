package models

import (
	"context"
	"errors"
	"fmt"

	yketypes "yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/mcclient"
	"yunion.io/yunioncloud/pkg/sqlchemy"

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

func (m *SNodeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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

func (m *SNodeManager) NewNode(data *apis.Node) (*SNode, error) {
	model, err := db.NewModelObject(m)
	if err != nil {
		return nil, err
	}

	node, ok := model.(*SNode)
	if !ok {
		return nil, fmt.Errorf("Convert to SNode error")
	}

	if data.ClusterId == "" {
		return nil, fmt.Errorf("ClusterId must provided")
	}

	if data.Name == "" {
		return nil, fmt.Errorf("Name must provided")
	}

	node.ClusterId = data.ClusterId
	node.Name = data.Name
	node.Etcd = data.Etcd
	node.ControlPlane = data.ControlPlane
	node.Worker = data.Worker
	node.NodeConfig = jsonutils.Marshal(data.NodeConfig)

	return node, nil
}

func (m *SNodeManager) Create(data *apis.Node) (*SNode, error) {
	node, err := m.NewNode(data)
	if err != nil {
		return nil, fmt.Errorf("New node by data %#v error: %v", data, err)
	}
	err = m.TableSpec().Insert(node)
	if err != nil {
		return nil, err
	}

	err = ClusterManager.AddClusterNodes(node.ClusterId, node)
	if err != nil {
		return nil, err
	}

	return node, nil
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

	ClusterId    string `nullable:"false" create:"required"`
	Address      string `nullable:"false"`
	Etcd         bool   `nullable:"true" default:"false"`
	ControlPlane bool   `nullable:"true" default:"false"`
	Worker       bool   `nullable:"true" default:"false"`

	NodeConfig jsonutils.JSONObject `nullable:"true"`
	DockerInfo jsonutils.JSONObject `nullable:"true"`
}

func (n *SNode) Node() (*apis.Node, error) {
	nodeConf := yketypes.ConfigNode{}
	if n.NodeConfig != nil {
		n.NodeConfig.Unmarshal(&nodeConf)
	}
	return &apis.Node{
		Name:         n.Name,
		Etcd:         n.Etcd,
		ControlPlane: n.ControlPlane,
		Worker:       n.Worker,
		NodeConfig:   &nodeConf,
	}, nil
}

func (n *SNode) Update(data *apis.Node) (*SNode, error) {
	_, err := n.GetModelManager().TableSpec().Update(n, func() error {
		//n.Name = data.Name
		n.Etcd = data.Etcd
		n.ControlPlane = data.ControlPlane
		n.Worker = data.Worker
		if data.DockerInfo != nil {
			dInfo := jsonutils.Marshal(data.DockerInfo)
			n.DockerInfo = dInfo
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = ClusterManager.UpdateCluster(n.ClusterId, n)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func (n *SNode) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return n.IsOwner(userCred)
}

func (n *SNode) ValidateDeleteCondition(ctx context.Context) error {
	// TODO: validate cluster status, only can delete when cluster ready
	return nil
}

func (n *SNode) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return nil
}
