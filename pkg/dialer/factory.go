package dialer

import (
	"yunion.io/yunion-kube/pkg/remotedialer"
	"yunion.io/yunion-kube/pkg/tunnelserver"
	"yunion.io/yunion-kube/pkg/types/config"
	"yunion.io/yunion-kube/pkg/types/config/dialer"
)

type Factory struct {
	Tunnelserver *remotedialer.Server
}

func NewFactory(apiCtx *config.ScaledContext) (dialer.Factory, error) {
	tunneler := tunnelserver.NewTunnelServer(apiCtx)
	return &Factory{
		Tunnelserver: tunneler,
	}, nil
}
