package clusters

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/models/manager"
)

type IClusterDriver interface {
	GetMode() apis.ModeType
	GetProvider() apis.ProviderType
	GetResourceType() apis.ClusterResourceType

	// GetK8sVersions return current cluster k8s versions supported
	GetK8sVersions() []string
	// GetUsableInstances return usable instances for cluster
	GetUsableInstances(session *mcclient.ClientSession) ([]apis.UsableInstance, error)
	// GetKubeconfig get current cluster kubeconfig
	GetKubeconfig(cluster *SCluster) (string, error)

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) error
	ValidateDeleteCondition() error
	ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine) error
	RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine, task taskman.ITask) error

	// CreateClusterResource create cluster resource to global k8s cluster
	CreateClusterResource(man *SClusterManager, data *apis.ClusterCreateInput) error
	ValidateCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, imageRepo *apis.ImageRepository, data []*apis.CreateMachineData) error
	// CreateMachines create machines record in db
	CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, data []*apis.CreateMachineData) ([]manager.IMachine, error)
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

	GetMachineDriver(resourceType apis.MachineResourceType) IMachineDriver
}

type IMachineDriver interface {
	ValidateCreateData(s *mcclient.ClientSession, input *apis.CreateMachineData) error
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
	mode apis.ModeType,
	provider apis.ProviderType,
	resType apis.ClusterResourceType,
) (IClusterDriver, error) {
	drv, err := clusterDrivers.Get(string(mode), string(provider), string(resType))
	if err != nil {
		return nil, err
	}
	return drv.(IClusterDriver), nil
}

func GetDriver(mode apis.ModeType, provider apis.ProviderType, resType apis.ClusterResourceType) IClusterDriver {
	drv, err := GetDriverWithError(mode, provider, resType)
	if err != nil {
		log.Fatalf("Get driver cluster provider: %s, resource type: %s error: %v", provider, resType, err)
	}
	return drv.(IClusterDriver)
}
