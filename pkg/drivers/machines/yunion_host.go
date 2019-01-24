package machines

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/utils"

	models "yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

const (
	HostTypeKVM     = "hypervisor"
	HostTypeKubelet = "kubelet"
)

type SYunionHostDriver struct {
}

func init() {
	driver := &SYunionHostDriver{}
	models.RegisterMachineDriver(driver)
}

func (d *SYunionHostDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeOnecloud
}

func (d *SYunionHostDriver) PrepareResource(session *mcclient.ClientSession, data *models.MachinePrepareData) (jsonutils.JSONObject, error) {
	hostId := data.InstanceId
	ret, err := cloudmod.Hosts.Get(session, hostId, nil)
	if err != nil {
		return nil, err
	}
	hostType, _ := ret.GetString("host_type")
	if !utils.IsInStringArray(hostType, []string{HostTypeKVM, HostTypeKubelet}) {
		return nil, fmt.Errorf("Host %q invalid host_type %q", hostId, hostType)
	}
	accessIP, _ := ret.GetString("access_ip")
	_, err = ssh.RemoteSSHBashScript("root", accessIP, "123@openmag", data.Script)
	return nil, err
}
