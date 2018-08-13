package ykenodeconfigserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/yke/pkg/services"
	"yunion.io/yke/pkg/types"

	"yunion.io/x/yunion-kube/pkg/libyke"
	"yunion.io/x/yunion-kube/pkg/tunnelserver"
	"yunion.io/x/yunion-kube/pkg/types/apis"
	"yunion.io/x/yunion-kube/pkg/types/config"
	"yunion.io/x/yunion-kube/pkg/types/slice"
	"yunion.io/x/yunion-kube/pkg/ykecerts"
	"yunion.io/x/yunion-kube/pkg/ykeworker"
)

type YKENodeConfigServer struct {
	auth   *tunnelserver.Authorizer
	lookup *ykecerts.BundleLookup
}

func Handler(auth *tunnelserver.Authorizer, scaledCtx *config.ScaledContext) http.Handler {
	return &YKENodeConfigServer{
		auth: auth,
		//lookup: ykecerts.NewLookup(),
	}
}

func (n *YKENodeConfigServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// 404 tells the client to continue without plan
	// 5xx tells the client to try again later for plan
	client, ok, err := n.auth.Authorize(req)
	if err != nil {
		httperrors.InternalServerError(rw, err.Error())
		return
	}

	if !ok {
		httperrors.UnauthorizedError(rw, "")
		return
	}

	if client.Node == nil {
		httperrors.NotFoundError(rw, "")
		return
	}

	log.Warningf("TODO: config server .....")
	//var nodeConfig *ykeworker.NodeConfig
	//if isNonWorkerOnly(client.Node.Role) {

	//}
}

func isNonWorkerOnly(role []string) bool {
	if slice.ContainsString(role, services.ETCDRole) ||
		slice.ContainsString(role, services.ControlRole) {
		return true
	}
	return false
}

func (n *YKENodeConfigServer) nonWorkerConfig(ctx context.Context, cluster *apis.Cluster, node *apis.Node) (*ykeworker.NodeConfig, error) {
	ykeConfig := cluster.YunionKubernetesEngineConfig
	if ykeConfig == nil {
		ykeConfig = &types.KubernetesEngineConfig{}
	}

	ykeConfig.Nodes = []types.ConfigNode{*node.NodeConfig}
	ykeConfig.Nodes[0].Role = []string{services.WorkerRole, services.ETCDRole, services.ControlRole}
	infos, err := libyke.GetDockerInfo(node)
	if err != nil {
		return nil, err
	}

	plan, err := libyke.New().GeneratePlan(ctx, ykeConfig, infos)
	if err != nil {
		return nil, err
	}

	nc := &ykeworker.NodeConfig{
		ClusterName: cluster.Name,
	}

	for _, tempNode := range plan.Nodes {
		if tempNode.Address == node.Address {
			nc.Processes = augmentProcesses(tempNode.Processes, false)
			return nc, nil
		}
	}
	return nil, fmt.Errorf("failed to find plan for non-worker %s", node.Address)
}

func (n *YKENodeConfigServer) nodeConfig(ctx context.Context, cluster *apis.Cluster, node *apis.Node) (*ykeworker.NodeConfig, error) {
	bundle, err := n.lookup.Lookup(cluster)
	if err != nil {
		return nil, err
	}

	//bundle = bundle.ForNode()
	log.Warningf("TODO nodeconfig, bundle: %v", bundle)
	return nil, nil
}

func augmentProcesses(processes map[string]types.Process, worker bool) map[string]types.Process {
	if worker {
		// not sure if we really need this anymore
		delete(processes, "etcd")
	}

	for _, p := range processes {
		for i, bind := range p.Binds {
			parts := strings.Split(bind, ":")
			if len(parts) > 1 && parts[1] == "/etc/kubernetes" {
				parts[0] = parts[1]
				p.Binds[i] = strings.Join(parts, ":")
			}
		}
	}
	return processes
}
