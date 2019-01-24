package machines

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/types"
)

type IMachineDriver interface {
	GetProvider() types.ProviderType
	PrepareResource(session *mcclient.ClientSession, machine *MachinePrepareData) (jsonutils.JSONObject, error)
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
	log.Fatalf("Unsupported provider: %s", provider)
	return nil
}
