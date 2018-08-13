package clusterrouter

import (
	"fmt"
	"net/http"

	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/yunion-kube/pkg/clusterrouter/proxy"
	"yunion.io/x/yunion-kube/pkg/models"
)

type factory struct {
}

func (s *factory) get(req *http.Request) (http.Handler, error) {
	clusterId := proxy.GetClusterId(req)
	if clusterId == "" {
		return nil, fmt.Errorf("ClusterId not provided by request: %#v", req)
	}
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(auth.GetTokenString(), clusterId)
	if err != nil {
		return nil, err
	}
	return s.newServer(cluster)
}

func (s *factory) newServer(c *models.SCluster) (*proxy.SRemoteService, error) {
	return proxy.New(c)
}
