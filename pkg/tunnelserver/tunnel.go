package tunnelserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
		log.Warningf("Header %s token not found", Token)
		return nil, false, nil
	}

	cluster, err := t.getClusterByToken(token)
	if err != nil || cluster == nil {
		return nil, false, err
	}

	input, err := t.readInput(cluster, req)
	if err != nil {
		log.Errorf("readInput error: %v", err)
		return nil, false, err
	}

	if input.Node == nil {
		return nil, false, nil
	}

	//register := strings.HasSuffix(req.URL.Path, "/register")
	node, ok, err := t.authorizeNode(cluster, input.Node, req)
	if err != nil {
		log.Errorf("authorizeNode error(%s): %v", req.RemoteAddr, err)
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

func (t *Authorizer) authorizeNode(cluster *apis.Cluster, inNode *client.Node, req *http.Request) (*apis.Node, bool, error) {
	node, err := t.getClusterNode(cluster, inNode)
	if err != nil {
		return nil, false, err
	}
	node, err = t.registerNode(node, inNode, req)
	if err != nil {
		return nil, false, fmt.Errorf("Register node error: %v", err)
	}

	apiNode, err := node.Node()
	return apiNode, true, err
}

func (t *Authorizer) registerNode(node *models.SNode, inNode *client.Node, req *http.Request) (*models.SNode, error) {
	if inNode.Address == "" {
		return nil, errors.New("invalid input, address empty")
	}

	data := &apis.Node{
		ClusterId:         node.ClusterId,
		RequestedHostname: inNode.RequestedHostname,
		Address:           inNode.Address,
		DockerInfo:        inNode.DockerInfo,
	}

	return node.Register(data)
}

func nodeIdent(node *client.Node) string {
	return node.Id
}

func (t *Authorizer) getClusterNode(cluster *apis.Cluster, inNode *client.Node) (*models.SNode, error) {
	nodeIdent := nodeIdent(inNode)
	node, err := t.NodeManager.FetchClusterNode(cluster.Id, nodeIdent)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (t *Authorizer) getClusterByToken(tokenId string) (*apis.Cluster, error) {
	return t.ClusterManager.GetClusterById(tokenId)
}

func (t *Authorizer) readInput(cluster *apis.Cluster, req *http.Request) (*input, error) {
	params := req.Header.Get(Params)
	var input input

	bytes, err := base64.StdEncoding.DecodeString(params)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytes, &input); err != nil {
		return nil, err
	}

	if input.Node == nil {
		return nil, errors.New("missing node registration info")
	}

	if input.Node.Id == "" {
		return nil, errors.New("missing node id")
	}

	if input.Node != nil && input.Node.RequestedHostname == "" {
		return nil, errors.New("invalid input, hostname empty")
	}

	if input.Node.Address == "" {
		return nil, errors.New("address empty")
	}

	return &input, nil
}
