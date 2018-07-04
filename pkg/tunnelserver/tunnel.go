package tunnelserver

import (
	"net/http"

	"yunion.io/yunion-kube/pkg/remotedialer"
	"yunion.io/yunion-kube/pkg/types/config"
)

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
