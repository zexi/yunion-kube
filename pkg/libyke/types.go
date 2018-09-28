package libyke

import (
	"context"

	dtypes "github.com/docker/docker/api/types"

	"yunion.io/x/yke/pkg/cluster"
	"yunion.io/x/yke/pkg/hosts"
	"yunion.io/x/yke/pkg/k8s"
	"yunion.io/x/yke/pkg/pki"
	"yunion.io/x/yke/pkg/types"
)

type YKE interface {
	GenerateYKENodeCerts(ctx context.Context, config types.KubernetesEngineConfig, nodeAddress string, certBundle map[string]pki.CertificatePKI) map[string]pki.CertificatePKI
	GenerateCerts(config *types.KubernetesEngineConfig) (map[string]pki.CertificatePKI, error)
	RegenerateEtcdCertificate(crtMap map[string]pki.CertificatePKI, etcdHost *hosts.Host, cluster *cluster.Cluster) (map[string]pki.CertificatePKI, error)
	ParseCluster(clusterName string, config *types.KubernetesEngineConfig, dockerDialerFactory, localConnDialerFactory hosts.DialerFactory, k8sWrapTransport k8s.WrapTransport) (*cluster.Cluster, error)
	GeneratePlan(ctx context.Context, config *types.KubernetesEngineConfig, dockerInfo map[string]dtypes.Info) (types.Plan, error)
}

func New() YKE {
	return &yke{}
}
