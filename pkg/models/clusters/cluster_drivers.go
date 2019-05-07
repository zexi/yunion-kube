package clusters

import (
	"context"

	providerv1 "yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type IClusterDriver interface {
	GetProvider() types.ProviderType
	GetResourceType() types.ClusterResourceType
	//UseClusterAPI() bool

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
	NeedGenerateCertificate() bool
}

type IClusterAPIDriver interface {
	IClusterDriver

	PreCreateClusterResource(s *mcclient.ClientSession, data *types.CreateClusterData, clusterSpec *providerv1.OneCloudClusterProviderSpec) error
	//PostCreateClusterResource() error
}

var clusterDrivers *drivers.DriverManager

func init() {
	clusterDrivers = drivers.NewDriverManager("")
}

func RegisterClusterDriver(driver IClusterDriver) {
	resType := driver.GetResourceType()
	provider := driver.GetProvider()
	err := clusterDrivers.Register(driver, string(provider), string(resType))
	if err != nil {
		log.Fatalf("cluster driver provider %s, resource type %s driver register error: %v", provider, resType, err)
	}
}

func GetDriver(provider types.ProviderType, resType types.ClusterResourceType) IClusterDriver {
	drv, err := clusterDrivers.Get(string(provider), string(resType))
	if err != nil {
		log.Fatalf("Get driver cluster provider: %s, resource type: %s error: %v", provider, resType, err)
	}
	return drv.(IClusterDriver)
}
