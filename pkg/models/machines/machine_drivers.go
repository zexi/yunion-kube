package machines

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
)

type IMachineDriver interface {
	clusters.IMachineDriver

	GetProvider() apis.ProviderType
	GetResourceType() apis.MachineResourceType
	GetPrivateIP(session *mcclient.ClientSession, resourceId string) (string, error)
	UseClusterAPI() bool

	PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *SMachine, data *jsonutils.JSONDict) error

	RequestPrepareMachine(ctx context.Context, userCred mcclient.TokenCredential, machine *SMachine, task taskman.ITask) error
	PrepareResource(session *mcclient.ClientSession, machine *SMachine, data *apis.MachinePrepareInput) (jsonutils.JSONObject, error)

	ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *SMachine) error
	TerminateResource(session *mcclient.ClientSession, machine *SMachine) error
}

var machineDrivers *drivers.DriverManager

func init() {
	machineDrivers = drivers.NewDriverManager("")
}

func RegisterMachineDriver(driver IMachineDriver) {
	resType := driver.GetResourceType()
	provider := driver.GetProvider()
	err := machineDrivers.Register(driver, string(provider), string(resType))
	if err != nil {
		log.Fatalf("machine driver provider %s, resource type %s driver register error: %v", provider, resType, err)
	}
}

func GetDriver(provider apis.ProviderType, resType apis.MachineResourceType) IMachineDriver {
	drv, err := machineDrivers.Get(string(provider), string(resType))
	if err != nil {
		log.Fatalf("Get machine driver provider: %s, resource type: %s error: %v", provider, resType, err)
	}
	return drv.(IMachineDriver)
}
