package machines

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type IMachineDriver interface {
	GetProvider() types.ProviderType
	GetPrivateIP(session *mcclient.ClientSession, id string) (string, error)

	GetKubeConfig(session *mcclient.ClientSession, machine *SMachine) (string, error)

	ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error
	PrepareResource(session *mcclient.ClientSession, machine *SMachine, data *MachinePrepareData) (jsonutils.JSONObject, error)
	ApplyAddons(cluster *clusters.SCluster, kubeconfig string) error
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
	log.Fatalf("Unsupported provider: %s", provider)
	return nil
}
