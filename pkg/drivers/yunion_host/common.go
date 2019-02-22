package yunion_host

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
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
	if m, err := machines.MachineManager.GetMachineByResourceId(hostId); err != nil {
		return nil, err
	} else if m != nil {
		return nil, httperrors.NewInputParameterError("Machine %s already use host %s", m.GetName(), hostId)
	}
	if !utils.IsInStringArray(hostType, []string{HostTypeKVM, HostTypeKubelet}) {
		return nil, httperrors.NewInputParameterError("Host %q invalid host_type %q", hostId, hostType)
	}
	return ret, nil
}

func GetV1Cluster(cluster *clusters.SCluster) (*models.SCluster, error) {
	return models.ClusterManager.FetchClusterByIdOrName(nil, cluster.GetName())
}

func GetV1Node(machine *machines.SMachine) (*models.SNode, error) {
	return models.NodeManager.FetchNodeByHostId(machine.ResourceId)
}

func GetControlplaneMachineDatas(cluster *clusters.SCluster, data []*types.CreateMachineData) ([]*types.CreateMachineData, []*types.CreateMachineData) {
	controls := make([]*types.CreateMachineData, 0)
	nodes := make([]*types.CreateMachineData, 0)
	for _, d := range data {
		if cluster != nil {
			d.ClusterId = cluster.GetId()
		}
		if d.Role == types.RoleTypeControlplane {
			controls = append(controls, d)
		} else {
			nodes = append(nodes, d)
		}
	}
	return controls, nodes
}

func validateCreateMachine(m *types.CreateMachineData) error {
	if err := machines.ValidateRole(m.Role); err != nil {
		return err
	}
	if err := ValidateResourceType(m.ResourceType); err != nil {
		return err
	}
	if len(m.ResourceId) == 0 {
		return httperrors.NewInputParameterError("ResourceId must provided")
	}
	session, err := clusters.ClusterManager.GetSession()
	if err != nil {
		return err
	}
	if _, err := ValidateHostId(session, m.ResourceId); err != nil {
		return err
	}
	return nil
}

func CheckControlplaneExists(cluster *clusters.SCluster) error {
	controlplane, err := cluster.GetRunningControlplaneMachine()
	if err != nil {
		return httperrors.NewInputParameterError("CheckControlplaneExists: %v", err)
	}
	if controlplane == nil {
		return fmt.Errorf("Running controlplane not exists")
	}
	return nil
}

func NeedControlplane(c *clusters.SCluster) (bool, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return false, err
	}
	if len(ms) == 0 {
		return true, nil
	}
	return false, nil
}

func ValidateAddMachines(c *clusters.SCluster, ms []*types.CreateMachineData) error {
	needControlplane, err := NeedControlplane(c)
	if err != nil {
		return err
	}
	controls, _ := GetControlplaneMachineDatas(c, ms)
	if needControlplane {
		if len(controls) == 0 {
			return httperrors.NewInputParameterError("controlplane node must created")
		}
	}

	//if !needControlplane {
	//if err := CheckControlplaneExists(c); err != nil {
	//return err
	//}
	//}

	for _, m := range ms {
		if err := validateCreateMachine(m); err != nil {
			return err
		}
	}
	return nil
}

func ValidateClusterCreateData(data *jsonutils.JSONDict) error {
	createData := types.CreateClusterData{}
	if err := data.Unmarshal(&createData); err != nil {
		return httperrors.NewInputParameterError("Unmarshal to CreateClusterData: %v", err)
	}
	ms := createData.Machines
	controls, _ := GetControlplaneMachineDatas(nil, ms)
	if len(controls) == 0 && createData.Provider != string(types.ProviderTypeSystem) {
		return httperrors.NewInputParameterError("No controlplane nodes")
	}
	for _, m := range ms {
		if err := validateCreateMachine(m); err != nil {
			return err
		}
	}
	return nil
}
