package clusters

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type IClusterDriver interface {
	GetMode() types.ModeType
	GetProvider() types.ProviderType
	GetResourceType() types.ClusterResourceType

	// GetK8sVersions return current cluster k8s versions supported
	GetK8sVersions() []string
	// GetUsableInstances return usable instances for cluster
	GetUsableInstances(session *mcclient.ClientSession) ([]types.UsableInstance, error)
	// GetKubeconfig get current cluster kubeconfig
	GetKubeconfig(cluster *SCluster) (string, error)

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error
	ValidateDeleteCondition() error
	ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine) error
	RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine, task taskman.ITask) error

	// CreateClusterResource create cluster resource to global k8s cluster
	CreateClusterResource(man *SClusterManager, data *types.CreateClusterData) error
	ValidateCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, data []*types.CreateMachineData) error
	// CreateMachines create machines record in db
	CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, data []*types.CreateMachineData) ([]manager.IMachine, error)
	// RequestDeployMachines deploy machines after machines created
	RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine, task taskman.ITask) error
	// GetAddonsManifest return addons yaml manifest to be applied to cluster
	GetAddonsManifest(cluster *SCluster) (string, error)
	// StartSyncStatus start cluster sync status task
	StartSyncStatus(cluster *SCluster, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error

	// need generate kubeadm certificates
	NeedGenerateCertificate() bool
	// NeedCreateMachines make this driver create machines models
	NeedCreateMachines() bool
}

var clusterDrivers *drivers.DriverManager

func init() {
	clusterDrivers = drivers.NewDriverManager("")
}

func RegisterClusterDriver(driver IClusterDriver) {
	modeType := driver.GetMode()
	resType := driver.GetResourceType()
	provider := driver.GetProvider()
	err := clusterDrivers.Register(driver,
		string(modeType),
		string(provider),
		string(resType))
	if err != nil {
		log.Fatalf("cluster driver provider %s, resource type %s driver register error: %v", provider, resType, err)
	}
}

func GetDriverWithError(
	mode types.ModeType,
	provider types.ProviderType,
	resType types.ClusterResourceType,
) (IClusterDriver, error) {
	drv, err := clusterDrivers.Get(string(mode), string(provider), string(resType))
	if err != nil {
		return nil, err
	}
	return drv.(IClusterDriver), nil
}

func GetDriver(mode types.ModeType, provider types.ProviderType, resType types.ClusterResourceType) IClusterDriver {
	drv, err := GetDriverWithError(mode, provider, resType)
	if err != nil {
		log.Fatalf("Get driver cluster provider: %s, resource type: %s error: %v", provider, resType, err)
	}
	return drv.(IClusterDriver)
}
