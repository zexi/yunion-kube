package ykedialerfactory

import (
	"fmt"
	"net"
	"strings"

	"yunion.io/yke/pkg/hosts"
	"yunion.io/yke/pkg/tunnel"
	"yunion.io/yunion-kube/pkg/types/config/dialer"
)

type YKEdialerfactory struct {
	Factory dialer.Factory
	Docker  bool
}

func (t *YKEdialerfactory) Build(h *hosts.Host) (func(network, address string) (net.Conn, error), error) {
	if h.NodeName == "" {
		return tunnel.SSHFactory(h.TunnelHostConfig())
	}

	parts := strings.SplitN(h.NodeName, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid name reference %q", h.NodeName)
	}

	//if t.Docker {
	return t.Factory.DockerDialer(parts[0], parts[1])
	//}
}
