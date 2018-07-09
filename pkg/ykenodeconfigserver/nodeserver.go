package ykenodeconfigserver

import (
	"net/http"

	"yunion.io/yunion-kube/pkg/ykecerts"
)

type YKENodeConfigServer struct {
	lookup *ykecerts.BundleLookup
}

func (n *YKENodeConfigServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

}
