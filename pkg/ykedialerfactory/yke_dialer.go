package ykedialerfactory

import (
	"fmt"
	"net/http"
	"strings"

	"yunion.io/yke/pkg/k8s"
	"yunion.io/yke/pkg/tunnel"
	"yunion.io/yke/pkg/types"

	"yunion.io/yunion-kube/pkg/types/config/dialer"
	"yunion.io/yunion-kube/pkg/types/slice"
)

type YKEDialerFactory struct {
	Factory dialer.Factory
	Docker  bool
}

func (t *YKEDialerFactory) Build(h tunnel.HostConfig) (tunnel.DialFunc, error) {
	if h.NodeName == "" {
		return tunnel.SSHFactory(h)
	}

	parts := strings.SplitN(h.NodeName, ":", 2)
	//if len(parts) != 2 {
	//return nil, fmt.Errorf("Invalid name reference %q", h.NodeName)
	//}
	clusterName, nodeName := "", ""
	if len(parts) == 1 {
		nodeName = parts[0]
	} else if len(parts) == 2 {
		clusterName = parts[0]
		nodeName = parts[1]
	} else {
		return nil, fmt.Errorf("Invalid name reference %q", h.NodeName)
	}

	if t.Docker {
		return t.Factory.DockerDialer(clusterName, nodeName)
	}
	return t.Factory.NodeDialer(clusterName, nodeName)
}

func (t *YKEDialerFactory) WrapTransport(config *types.KubernetesEngineConfig) k8s.WrapTransport {
	parse := func(ref string) (clusterName, nodeName string) {
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

		clusterName, nodeName := parse(node.NodeName)
		dialer, err := t.Factory.NodeDialer(clusterName, nodeName)
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
