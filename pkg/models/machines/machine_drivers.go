package machines

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/types"
)

type IMachineDriver interface {
	GetProvider() types.ProviderType
	GetPrivateIP(session *mcclient.ClientSession, id string) (string, error)
	UseClusterAPI() bool

	ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error
	PrepareResource(session *mcclient.ClientSession, machine *SMachine, data *MachinePrepareData) (jsonutils.JSONObject, error)
	PostDelete(machine *SMachine) error
	TerminateResource(session *mcclient.ClientSession, machine *SMachine) error
}

var machineDrivers map[types.ProviderType]IMachineDriver

func init() {
	machineDrivers = make(map[types.ProviderType]IMachineDriver)
}

func RegisterMachineDriver(driver IMachineDriver) {
	machineDrivers[driver.GetProvider()] = driver
}

func GetDriver(provider types.ProviderType) IMachineDriver {
	driver, ok := machineDrivers[provider]
	if ok {
		return driver
	}
	log.Fatalf("Unsupported machine provider: %s", provider)
	return nil
}
