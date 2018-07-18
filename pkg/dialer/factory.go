package dialer

import (
	"fmt"
	"net"
	"time"

	"yunion.io/yke/pkg/tunnel"

	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/remotedialer"
	"yunion.io/yunion-kube/pkg/tunnelserver"
	"yunion.io/yunion-kube/pkg/types/config"
	"yunion.io/yunion-kube/pkg/types/config/dialer"
)

type Factory struct {
	Tunnelserver     *remotedialer.Server
	TunnelAuthorizer *tunnelserver.Authorizer
	ClusterManager   *models.SClusterManager
	NodeManager      *models.SNodeManager
}

func NewFactory(apiCtx *config.ScaledContext) (dialer.Factory, error) {
	authorizer := tunnelserver.NewAuthorizer(apiCtx)
	tunneler := tunnelserver.NewTunnelServer(apiCtx, authorizer)
	return &Factory{
		Tunnelserver:     tunneler,
		TunnelAuthorizer: authorizer,
		ClusterManager:   apiCtx.ClusterManager,
		NodeManager:      apiCtx.NodeManager,
	}, nil
}

func (f *Factory) DockerDialer(clusterId, nodeId string) (tunnel.DialFunc, error) {
	node, err := f.NodeManager.FetchClusterNode(clusterId, nodeId)
	if err != nil {
		return nil, err
	}
	machineName := node.Name
	if f.Tunnelserver.HasSession(machineName) {
		log.Warningf("=======docker dialer: %s", machineName)
		d := f.Tunnelserver.Dialer(machineName, 15*time.Second)
		return func(string, string) (net.Conn, error) {
			return d("unix", "/var/run/docker.sock")
		}, nil
	}

	return nil, fmt.Errorf("can not build docker dialer to %s:%s", clusterId, machineName)
}

func (f *Factory) NodeDialer(clusterId, nodeId string) (tunnel.DialFunc, error) {
	return func(network, address string) (net.Conn, error) {
		d, err := f.nodeDialer(clusterId, nodeId)
		if err != nil {
			return nil, err
		}
		return d(network, address)
	}, nil
}

func (f *Factory) nodeDialer(clusterId, nodeId string) (tunnel.DialFunc, error) {
	node, err := f.NodeManager.FetchClusterNode(clusterId, nodeId)
	if err != nil {
		return nil, err
	}
	machineName := node.Name

	if f.Tunnelserver.HasSession(machineName) {
		d := f.Tunnelserver.Dialer(machineName, 15*time.Second)
		return tunnel.DialFunc(d), nil
	}

	return nil, fmt.Errorf("can not build node dialer to %s:%s", clusterId, machineName)
}
