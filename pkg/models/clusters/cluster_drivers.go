package clusters

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type IClusterDriver interface {
	GetProvider() types.ProviderType
	UseClusterAPI() bool

	// GetK8sVersions return current cluster k8s versions supported
	GetK8sVersions() []string
	// GetUsableInstances return usable instances for cluster
	GetUsableInstances(session *mcclient.ClientSession) ([]types.UsableInstance, error)
	// GetKubeconfig get current cluster kubeconfig
	GetKubeconfig(cluster *SCluster) (string, error)

	ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error
	ValidateDeleteCondition() error
	ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine) error
	RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine) error

	// CreateClusterResource create cluster resource to global k8s cluster
	CreateClusterResource(man *SClusterManager, data *types.CreateClusterData) error
	ValidateAddMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, data []*types.CreateMachineData) error
	// RequestCreateMachines create machines after cluster created
	RequestCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, data []*types.CreateMachineData, task taskman.ITask) error
	// RequestDeleteCluster delete cluster when cluster delete
	RequestDeleteCluster(cluster *SCluster) error
	// ValidateAddMachine validate create machine resource
	ValidateAddMachine(cluster *SCluster, machine *types.CreateMachineData) error
	// GetAddonsManifest return addons yaml manifest to be applied to cluster
	GetAddonsManifest(cluster *SCluster) (string, error)
	// StartSyncStatus start cluster sync status task
	StartSyncStatus(cluster *SCluster, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error
}

var clusterDrivers map[types.ProviderType]IClusterDriver

func init() {
	clusterDrivers = make(map[types.ProviderType]IClusterDriver)
}

func RegisterClusterDriver(driver IClusterDriver) {
	clusterDrivers[driver.GetProvider()] = driver
}

func GetDriver(provider types.ProviderType) IClusterDriver {
	driver, ok := clusterDrivers[provider]
	if ok {
		return driver
	}
	log.Fatalf("Unsupported cluster provider: %s", provider)
	return nil
}
