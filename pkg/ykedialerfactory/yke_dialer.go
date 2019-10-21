package ykedialerfactory

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"yunion.io/x/yke/pkg/hosts"
	"yunion.io/x/yke/pkg/k8s"
	"yunion.io/x/yke/pkg/types"

	"yunion.io/x/yunion-kube/pkg/types/config/dialer"
	"yunion.io/x/yunion-kube/pkg/types/slice"
)

type YKEDialerFactory struct {
	Factory dialer.Factory
	Docker  bool
}

func (t *YKEDialerFactory) Build(h *hosts.Host) (func(network, address string) (net.Conn, error), error) {
	if h.NodeName == "" {
		return hosts.SSHFactory(h)
	}

	parts := strings.SplitN(h.NodeName, ":", 2)
	//if len(parts) != 2 {
	//return nil, fmt.Errorf("Invalid name reference %q", h.NodeName)
	//}
	clusterId, nodeId := "", ""
	if len(parts) == 1 {
		nodeId = parts[0]
	} else if len(parts) == 2 {
		clusterId = parts[0]
		nodeId = parts[1]
	} else {
		return nil, fmt.Errorf("Invalid name reference %q", h.NodeName)
	}

	if t.Docker {
		return t.Factory.DockerDialer(clusterId, nodeId)
	}
	return t.Factory.NodeDialer(clusterId, nodeId)
}

func (t *YKEDialerFactory) WrapTransport(config *types.KubernetesEngineConfig) k8s.WrapTransport {
	parse := func(ref string) (clusterId, nodeId string) {
		parts := strings.SplitN(ref, ":", 2)
		if len(parts) == 1 {
			return "", parts[0]
		}
		return parts[0], parts[1]
	}
	for _, node := range config.Nodes {
		if !slice.ContainsString(node.Role, "controlplane") {
			continue
		}

		clusterId, nodeId := parse(node.NodeName)
		dialer, err := t.Factory.NodeDialer(clusterId, nodeId)
		if dialer == nil || err != nil {
			continue
		}

		return func(rt http.RoundTripper) http.RoundTripper {
			if ht, ok := rt.(*http.Transport); ok {
				ht.DialContext = nil
				ht.DialTLS = nil
				ht.Dial = dialer
			}
			return rt
		}
	}

	return nil
}
