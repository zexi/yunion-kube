package machines

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type IMachineDriver interface {
	GetProvider() types.ProviderType
	GetResourceType() types.MachineResourceType
	GetPrivateIP(session *mcclient.ClientSession, id string) (string, error)
	UseClusterAPI() bool

	ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error
	PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *SMachine, data *jsonutils.JSONDict) error
	PrepareResource(session *mcclient.ClientSession, machine *SMachine, data *MachinePrepareData) (jsonutils.JSONObject, error)
	ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *SMachine) error
	PostDelete(ctx context.Context, userCred mcclient.TokenCredential, machine *SMachine, task taskman.ITask) error
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

func GetDriver(provider types.ProviderType, resType types.MachineResourceType) IMachineDriver {
	drv, err := machineDrivers.Get(string(provider), string(resType))
	if err != nil {
		log.Fatalf("Get machine driver provider: %s, resource type: %s error: %v", provider, resType, err)
	}
	return drv.(IMachineDriver)
}
