package dialer

import (
	"fmt"
	"net"
	"time"

	"yunion.io/yke/pkg/tunnel"

	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/remotedialer"
	"yunion.io/yunion-kube/pkg/tunnelserver"
	"yunion.io/yunion-kube/pkg/types/config"
	"yunion.io/yunion-kube/pkg/types/config/dialer"
)

type Factory struct {
	Tunnelserver     *remotedialer.Server
	TunnelAuthorizer *tunnelserver.Authorizer
}

func NewFactory(apiCtx *config.ScaledContext) (dialer.Factory, error) {
	authorizer := tunnelserver.NewAuthorizer(apiCtx)
	tunneler := tunnelserver.NewTunnelServer(apiCtx, authorizer)
	return &Factory{
		Tunnelserver:     tunneler,
		TunnelAuthorizer: authorizer,
	}, nil
}

func (f *Factory) DockerDialer(clusterName, machineName string) (tunnel.DialFunc, error) {
	if f.Tunnelserver.HasSession(machineName) {
		d := f.Tunnelserver.Dialer(machineName, 15*time.Second)
		return func(string, string) (net.Conn, error) {
			return d("unix", "/var/run/docker.sock")
		}, nil
	}

	return nil, fmt.Errorf("can not build dialer to %s:%s", clusterName, machineName)
}

func (f *Factory) NodeDialer(clusterName, machineName string) (tunnel.DialFunc, error) {
	return func(network, address string) (net.Conn, error) {
		d, err := f.nodeDialer(clusterName, machineName)
		if err != nil {
			return nil, err
		}
		return d(network, address)
	}, nil
}

func (f *Factory) nodeDialer(clusterName, machineName string) (tunnel.DialFunc, error) {
	machine, err := models.NodeManager.GetNode(clusterName, machineName)
	if err != nil {
		return nil, err
	}

	if f.Tunnelserver.HasSession(machine.Name) {
		d := f.Tunnelserver.Dialer(machine.Name, 15*time.Second)
		return tunnel.DialFunc(d), nil
	}

	return nil, fmt.Errorf("can not build dialer to %s:%s", clusterName, machineName)
}
