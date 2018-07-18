package libyke

import (
	"context"

	dtypes "github.com/docker/docker/api/types"

	"yunion.io/yke/pkg/cluster"
	"yunion.io/yke/pkg/hosts"
	"yunion.io/yke/pkg/k8s"
	"yunion.io/yke/pkg/pki"
	"yunion.io/yke/pkg/tunnel"
	"yunion.io/yke/pkg/types"

	"yunion.io/yunion-kube/pkg/types/apis"
	"yunion.io/yunion-kube/pkg/utils/convert"
)

type yke struct{}

func (*yke) GenerateYKENodeCerts(ctx context.Context, config types.KubernetesEngineConfig, nodeAddress string, certBundle map[string]pki.CertificatePKI) map[string]pki.CertificatePKI {
	return pki.GenerateNodeCerts(ctx, config, nodeAddress, certBundle)
}

func (*yke) GenerateCerts(config *types.KubernetesEngineConfig) (map[string]pki.CertificatePKI, error) {
	return pki.GenerateKECerts(context.Background(), *config, "", "")
}

func (*yke) RegenerateEtcdCertificate(crtMap map[string]pki.CertificatePKI, etcdHost *hosts.Host, cluster *cluster.Cluster) (map[string]pki.CertificatePKI, error) {
	return pki.RegenerateEtcdCertificate(context.Background(),
		crtMap,
		etcdHost,
		cluster.EtcdHosts,
		cluster.ClusterDomain,
		cluster.KubernetesServiceIP)
}

func (*yke) ParseCluster(clusterName string, config *types.KubernetesEngineConfig, dockerDialerFactory, localConnDialerFactory tunnel.DialerFactory, k8sWrapTransport k8s.WrapTransport) (*cluster.Cluster, error) {
	clusterFilePath := clusterName + "-cluster.yaml"
	if clusterName == "local" {
		clusterFilePath = ""
	}
	return cluster.ParseCluster(context.Background(),
		config, clusterFilePath, "",
		dockerDialerFactory, localConnDialerFactory,
		k8sWrapTransport)
}

func (*yke) GeneratePlan(ctx context.Context, config *types.KubernetesEngineConfig, dockerInfo map[string]dtypes.Info) (types.Plan, error) {
	return cluster.GeneratePlan(ctx, config)
}

func GetDockerInfo(node *apis.Node) (map[string]dtypes.Info, error) {
	infos := map[string]dtypes.Info{}
	if node.DockerInfo != nil {
		dockerInfo := dtypes.Info{}
		err := convert.ToObj(node.DockerInfo, &dockerInfo)
		if err != nil {
			return nil, err
		}
		infos[node.Address] = dockerInfo
	}
	return infos, nil
}
