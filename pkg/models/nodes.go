package models

import (
	"context"
	"errors"
	"fmt"

	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/mcclient"

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

func (m *SNodeManager) FetchNode(ident string) *SNode {
	node, err := m.FetchByIdOrName("", ident)
	if err != nil {
		log.Errorf("Fetch node %q fail: %v", ident, err)
		return nil
	}
	return node.(*SNode)
}

func (m *SNodeManager) GetNode(ident string) (*apis.Node, error) {
	node := m.FetchNode(ident)
	if node == nil {
		return nil, NodeNotFoundError
	}
	return node.Node()
}

func (m *SNodeManager) Create(data *apis.Node) (*SNode, error) {
	model, err := db.NewModelObject(m)
	if err != nil {
		return nil, err
	}
	node, ok := model.(*SNode)
	if !ok {
		return nil, fmt.Errorf("Convert to SNode error")
	}
	node.Name = data.Name
	node.Etcd = data.Etcd
	node.ControlPlane = data.ControlPlane
	node.Worker = data.Worker
	err = m.TableSpec().Insert(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (m *SNodeManager) Update(data *apis.Node) (*SNode, error) {
	node := m.FetchNode(data.Name)
	if node == nil {
		return nil, NodeNotFoundError
	}
	return node.Update(data)
}

type SNode struct {
	db.SVirtualResourceBase
	ClusterId    string `nullable:"false" create:"required"`
	Etcd         bool   `nullable:"true" default:"false"`
	ControlPlane bool   `nullable:"true" default:"false"`
	Worker       bool   `nullable:"true" default:"false"`

	CustomConfig jsonutils.JSONObject `nullable:"true" list:"admin"`
	DockerInfo   jsonutils.JSONObject `nullable:"true"`
}

func (n *SNode) Node() (*apis.Node, error) {
	conf := apis.CustomConfig{}
	if n.CustomConfig != nil {
		n.CustomConfig.Unmarshal(&conf)
	}
	return &apis.Node{
		Name:         n.Name,
		Etcd:         n.Etcd,
		ControlPlane: n.ControlPlane,
		Worker:       n.Worker,
		CustomConfig: &conf,
	}, nil
}

func (n *SNode) Update(data *apis.Node) (*SNode, error) {
	_, err := n.GetModelManager().TableSpec().Update(n, func() error {
		n.Name = data.Name
		n.Etcd = data.Etcd
		n.ControlPlane = data.ControlPlane
		conf := jsonutils.Marshal(data.CustomConfig)
		n.CustomConfig = conf
		dInfo := jsonutils.Marshal(data.DockerInfo)
		n.DockerInfo = dInfo
		return nil
	})
	return n, err
}
