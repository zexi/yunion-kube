package node

import (
	"context"
	"os"
	"strings"

	"github.com/docker/docker/client"

	"yunion.io/yunion-kube/pkg/types"
	"yunion.io/yunion-kube/pkg/types/slice"
	"yunion.io/yunioncloud/pkg/log"
)

func TokenAndURL() (string, string, error) {
	return os.Getenv(types.ENV_AGENT_TOKEN), os.Getenv(types.ENV_AGENT_SERVER), nil
}

func Params() map[string]interface{} {
	roles := split(os.Getenv(types.ENV_AGENT_ROLE))
	params := map[string]interface{}{
		"customConfig": map[string]interface{}{
			"address":         os.Getenv(types.ENV_AGENT_ADDRESS),
			"internalAddress": os.Getenv(types.ENV_AGENT_INTERNAL_ADDRESS),
			"roles":           split(os.Getenv(types.ENV_AGENT_ROLE)),
		},
		"etcd":              slice.ContainsString(roles, types.ROLE_ETCD),
		"controlPlane":      slice.ContainsString(roles, types.ROLE_CONTROL_PLANE),
		"worker":            slice.ContainsString(roles, types.ROLE_WORKER),
		"requestedHostname": os.Getenv(types.ENV_AGENT_NODE_NAME),
	}

	for k, v := range params {
		if m, ok := v.(map[string]string); ok {
			for k, v := range m {
				log.Infof("Option %s=%s", k, v)
			}
		} else {
			log.Infof("Option %s=%v", k, v)
		}
	}

	dclient, err := client.NewEnvClient()
	if err == nil {
		info, err := dclient.Info(context.Background())
		if err == nil {
			params["dockerInfo"] = info
		}
	}

	return map[string]interface{}{
		"node": params,
	}
}

func split(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 1 && result[0] == "" {
		return nil
	}
	return result
}
