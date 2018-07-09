package dialer

import (
	"fmt"
	"net"
	"time"

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

func (f *Factory) DockerDialer(clusterName, machineName string) (dialer.Dialer, error) {
	if f.Tunnelserver.HasSession(machineName) {
		d := f.Tunnelserver.Dialer(machineName, 15*time.Second)
		return func(string, string) (net.Conn, error) {
			return d("unix", "/var/run/docker.sock")
		}, nil
	}

	return nil, fmt.Errorf("can not build dialer to %s:%s", clusterName, machineName)
}
