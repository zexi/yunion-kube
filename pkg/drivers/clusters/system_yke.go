package clusters

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

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

func (d *SSystemYKEDriver) RequestCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, data []*types.CreateMachineData, task taskman.ITask) error {
	return httperrors.NewUnsupportOperationError("Global system cluster can't be machines")
}

func (d *SSystemYKEDriver) ValidateDeleteCondition() error {
	return httperrors.NewUnsupportOperationError("Global system cluster can't be delete")
}
