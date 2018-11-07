package dialer

import (
	"net"
)

type Dialer func(network, address string) (net.Conn, error)

type Factory interface {
	//LocalClusterDialer() Dialer
	//ClusterDialer(clusterName string) (Dialer, error)
	DockerDialer(clusterId, nodeId string) (Dialer, error)
	NodeDialer(clusterId, nodeId string) (Dialer, error)
}
