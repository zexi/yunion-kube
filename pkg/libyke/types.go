package libyke

import (
	"context"

	"yunion.io/yke/pkg/cluster"
	"yunion.io/yke/pkg/hosts"
	"yunion.io/yke/pkg/k8s"
	"yunion.io/yke/pkg/pki"
	"yunion.io/yke/pkg/tunnel"
	"yunion.io/yke/pkg/types"
)

type YKE interface {
	GenerateYKENodeCerts(ctx context.Context, config types.KubernetesEngineConfig, nodeAddress string, certBundle map[string]pki.CertificatePKI) map[string]pki.CertificatePKI
	GenerateCerts(config *types.KubernetesEngineConfig) (map[string]pki.CertificatePKI, error)
	RegenerateEtcdCertificate(crtMap map[string]pki.CertificatePKI, etcdHost *hosts.Host, cluster *cluster.Cluster) (map[string]pki.CertificatePKI, error)
	ParseCluster(clusterName string, config *types.KubernetesEngineConfig, dockerDialerFactory, localConnDialerFactory tunnel.DialerFactory, k8sWrapTransport k8s.WrapTransport) (*cluster.Cluster, error)
	GeneratePlan(ctx context.Context, config *types.KubernetesEngineConfig, dockerInfo map[string]types.Info) (types.Plan, error)
}

func New() YKE {
	return &yke{}
}
