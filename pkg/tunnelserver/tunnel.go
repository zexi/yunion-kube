package tunnelserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	yketypes "yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/remotedialer"
	"yunion.io/yunion-kube/pkg/types/apis"
	"yunion.io/yunion-kube/pkg/types/client"
	"yunion.io/yunion-kube/pkg/types/config"
)

const (
	Token  = "X-API-Tunnel-Token"
	Params = "X-API-Tunnel-Params"
)

type input struct {
	Node *client.Node `json:"node"`
}

type Client struct {
	Cluster *apis.Cluster
	Node    *apis.Node
	Server  string
}

func NewTunnelServer(ctx *config.ScaledContext, authorizer *Authorizer) *remotedialer.Server {
	ready := func() bool {
		return true
	}
	return remotedialer.New(authorizer.authorizeTunnel, func(rw http.ResponseWriter, req *http.Request, code int, err error) {
		rw.WriteHeader(code)
		rw.Write([]byte(err.Error()))
	}, ready)
}

type Authorizer struct {
	ClusterManager *models.SClusterManager
	NodeManager    *models.SNodeManager
}

func NewAuthorizer(context *config.ScaledContext) *Authorizer {
	auth := &Authorizer{
		ClusterManager: models.ClusterManager,
		NodeManager:    models.NodeManager,
	}
	return auth
}

func (t *Authorizer) authorizeTunnel(req *http.Request) (string, bool, error) {
	client, ok, err := t.Authorize(req)
	if client != nil && client.Node != nil {
		return client.Node.Name, ok, err
	}
	return "", false, err
}

func (t *Authorizer) Authorize(req *http.Request) (*Client, bool, error) {
	token := req.Header.Get(Token)
	if token == "" {
		return nil, false, nil
	}

	cluster, err := t.getClusterByToken(token)
	if err != nil || cluster == nil {
		return nil, false, err
	}
	log.Debugf("Get cluster: %#v", cluster)

	input, err := t.readInput(cluster, req)
	if err != nil {
		log.Errorf("readInput error: %v", err)
		return nil, false, err
	}
	log.Debugf("Get input: %#v", input)

	if input.Node == nil {
		return nil, false, nil
	}

	register := strings.HasSuffix(req.URL.Path, "/register")
	node, ok, err := t.authorizeNode(register, cluster, input.Node, req)
	if err != nil {
		log.Errorf("authorizeNode error: %v", err)
		return nil, false, err
	}

	return &Client{
		Cluster: cluster,
		Node:    node,
		Server:  req.Host,
	}, ok, err
}

func IsNodeNotFound(err error) bool {
	return err == models.NodeNotFoundError
}

func (t *Authorizer) authorizeNode(register bool, cluster *apis.Cluster, inNode *client.Node, req *http.Request) (*apis.Node, bool, error) {
	node, err := t.getClusterNode(cluster, inNode)
	if IsNodeNotFound(err) {
		if !register {
			return nil, false, err
		}
		node, err = t.createNode(inNode, cluster, req)
		if err != nil {
			return nil, false, fmt.Errorf("Create node: %v", err)
		}
	} else if err != nil && node == nil {
		return nil, false, err
	}

	if register {
		log.Debugf("==== register coming")
		node, err = t.updateNode(node, inNode, cluster)
		if err != nil {
			return nil, false, err
		}
	}

	apiNode, err := node.Node()
	return apiNode, true, err
}

func (t *Authorizer) createNode(inNode *client.Node, cluster *apis.Cluster, req *http.Request) (*models.SNode, error) {
	customConfig := inNode.CustomConfig
	if customConfig == nil {
		return nil, errors.New("invalid input, mssing custom config")
	}

	if customConfig.Address == "" {
		return nil, errors.New("invalid input, address empty")
	}

	nodeConfig := clientNodeConfig(inNode)

	name := nodeName(inNode)

	node := &apis.Node{
		ClusterId:         cluster.Id,
		Name:              name,
		Etcd:              inNode.Etcd,
		ControlPlane:      inNode.ControlPlane,
		Worker:            inNode.Worker,
		RequestedHostname: inNode.RequestedHostname,
		NodeConfig:        nodeConfig,
	}

	return t.NodeManager.Create(node)
}

func clientNodeConfig(inNode *client.Node) *yketypes.ConfigNode {
	customConfig := inNode.CustomConfig
	if customConfig == nil {
		return nil
	}
	return &yketypes.ConfigNode{
		Role:    customConfig.Roles,
		Address: customConfig.Address,
	}
}

func getNodeFromClient(inNode *client.Node) *apis.Node {
	node := &apis.Node{}
	//node.Name = inNode.Name
	node.Etcd = inNode.Etcd
	node.ControlPlane = inNode.ControlPlane
	node.Worker = inNode.Worker
	node.RequestedHostname = inNode.RequestedHostname
	node.CustomConfig = inNode.CustomConfig
	node.DockerInfo = inNode.DockerInfo
	node.NodeConfig = clientNodeConfig(inNode)
	return node
}

func (t *Authorizer) updateNode(obj *models.SNode, inNode *client.Node, cluster *apis.Cluster) (*models.SNode, error) {
	node := getNodeFromClient(inNode)
	return obj.Update(node)
}

func nodeName(node *client.Node) string {
	return node.RequestedHostname
}

func (t *Authorizer) getClusterNode(cluster *apis.Cluster, inNode *client.Node) (*models.SNode, error) {
	nodeName := nodeName(inNode)
	node := t.NodeManager.FetchClusterNode(cluster.Name, nodeName)
	if node == nil {
		return nil, models.NodeNotFoundError
	}
	return node, nil
}

func (t *Authorizer) getClusterByToken(tokenId string) (*apis.Cluster, error) {
	return t.ClusterManager.GetCluster(tokenId)
}

func (t *Authorizer) readInput(cluster *apis.Cluster, req *http.Request) (*input, error) {
	params := req.Header.Get(Params)
	var input input

	bytes, err := base64.StdEncoding.DecodeString(params)
	if err != nil {
		return nil, err
	}
	log.Debugf("Get input params: %s", string(bytes))

	if err := json.Unmarshal(bytes, &input); err != nil {
		return nil, err
	}

	if input.Node == nil {
		return nil, errors.New("missing node registration info")
	}

	if input.Node != nil && input.Node.RequestedHostname == "" {
		return nil, errors.New("invalid input, hostname empty")
	}

	return &input, nil
}
