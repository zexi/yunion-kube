package clusters

import (
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type SSystemYKEDriver struct {
	*sBaseDriver
}

func NewSystemYKEDriver() *SSystemYKEDriver {
	return &SSystemYKEDriver{
		sBaseDriver: newBaseDriver(),
	}
}

func init() {
	clusters.RegisterClusterDriver(NewSystemYKEDriver())
}

func (d *SSystemYKEDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeSystem
}

func (d *SSystemYKEDriver) GetKubeconfig(cluster *clusters.SCluster) (string, error) {
	c, err := models.ClusterManager.FetchClusterByIdOrName(nil, cluster.GetName())
	if err != nil {
		return "", err
	}
	return c.GetAdminKubeconfig()
}

func (d *SSystemYKEDriver) ValidateDeleteCondition() error {
	return httperrors.NewUnsupportOperationError("Global system cluster can't be delete")
}
