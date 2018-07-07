package tunnelserver

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"yunion.io/yunion-kube/pkg/remotedialer"
	"yunion.io/yunion-kube/pkg/types"
	"yunion.io/yunion-kube/pkg/types/config"
)

const (
	Token  = "X-API-Tunnel-Token"
	Params = "X-API-Tunnel-Params"
)

type cluster struct {
	Address string `json:"address"`
	Token   string `json:"token"`
	CACert  string `json:"caCert"`
}

type input struct {
	//Node *client.Node `json:"node"`
	Cluster *cluster `json:"cluster"`
}

func NewTunnelServer(ctx *config.ScaledContext) *remotedialer.Server {
	ready := func() bool {
		return true
	}
	authorizer := func(req *http.Request) (string, bool, error) {
		id := req.Header.Get("x-tunnel-id")
		return id, id != "", nil
	}
	return remotedialer.New(authorizer, func(rw http.ResponseWriter, req *http.Request, code int, err error) {
		rw.WriteHeader(code)
		rw.Write([]byte(err.Error()))
	}, ready)
}

type Authorizer struct {
}

func NewAuthorizer(context *config.ScaledContext) *Authorizer {
	auth := &Authorizer{}
	return auth
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

	input, err := t.readInput(cluster, req)
}

func (t *Authorizer) getClusterByToken(token string) (*types.Cluster, error) {
	return nil, nil
}

func (t *Authorizer) readInput(cluster *types.Cluster, req *http.Request) (*input, error) {
	params := req.Header.Get(Params)
	var input input

	bytes, err := base64.StdEncoding.DecodeString(params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &input); err != nil {
		return nil, err
	}
}
