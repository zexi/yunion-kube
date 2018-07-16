package dialer

import (
	"net"

	"yunion.io/yke/pkg/tunnel"
)

type Dialer func(network, address string) (net.Conn, error)

type Factory interface {
	//LocalClusterDialer() Dialer
	//ClusterDialer(clusterName string) (Dialer, error)
	DockerDialer(clusterName, machineName string) (tunnel.DialFunc, error)
	NodeDialer(clusterName, machineName string) (tunnel.DialFunc, error)
}
