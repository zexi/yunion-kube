package clusterrouter

import (
	"net/http"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type Router struct {
	serverFactory *factory
}

func New() *Router {
	return &Router{serverFactory: &factory{}}
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	handler, err := r.serverFactory.get(req)
	if err != nil {
		log.Errorf("Router get server error: %v", err)
		httperrors.GeneralServerError(rw, err)
		return
	}

	handler.ServeHTTP(rw, req)
}
