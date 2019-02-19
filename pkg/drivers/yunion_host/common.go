package yunion_host

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

const (
	HostTypeKVM     = "hypervisor"
	HostTypeKubelet = "kubelet"
)

func ValidateResourceType(resType string) error {
	if resType != types.MachineResourceTypeBaremetal {
		return httperrors.NewInputParameterError("Invalid resource type: %q", resType)
	}
	return nil
}

func ValidateHostId(s *mcclient.ClientSession, hostId string) (jsonutils.JSONObject, error) {
	ret, err := cloudmod.Hosts.Get(s, hostId, nil)
	if err != nil {
		return nil, err
	}
	hostType, _ := ret.GetString("host_type")
	hostId, _ = ret.GetString("id")
	if m := machines.MachineManager.GetMachineByResourceId(hostId); m != nil {
		return nil, httperrors.NewInputParameterError("Machine %s already use host %s", m.GetName(), hostId)
	}
	if !utils.IsInStringArray(hostType, []string{HostTypeKVM, HostTypeKubelet}) {
		return nil, httperrors.NewInputParameterError("Host %q invalid host_type %q", hostId, hostType)
	}
	return ret, nil
}
