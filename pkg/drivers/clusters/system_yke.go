package clusters

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/drivers/system_yke"
	"yunion.io/x/yunion-kube/pkg/drivers/yunion_host"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type SSystemYKEDriver struct {
	*sBaseDriver
}

func NewSystemYKEDriver() *SSystemYKEDriver {
	return &SSystemYKEDriver{
		sBaseDriver: newBaseDriver(),
	}
}

func init() {
	clusters.RegisterClusterDriver(NewSystemYKEDriver())
}

func (d *SSystemYKEDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeSystem
}

func (d *SSystemYKEDriver) GetResourceType() types.ClusterResourceType {
	return types.ClusterResourceTypeHost
}

func (d *SSystemYKEDriver) GetK8sVersions() []string {
	return []string{
		models.DEFAULT_K8S_VERSION,
	}
}

func GetUsableCloudHosts(s *mcclient.ClientSession) ([]types.UsableInstance, error) {
	params := jsonutils.NewDict()
	filter := jsonutils.NewArray()
	filter.Add(jsonutils.NewString(fmt.Sprintf("host_type.in(%s, %s)", "hypervisor", "kubelet")))
	filter.Add(jsonutils.NewString("host_status.equals(online)"))
	filter.Add(jsonutils.NewString("status.equals(running)"))
	params.Add(filter, "filter")
	result, err := cloudmod.Hosts.List(s, params)
	if err != nil {
		return nil, err
	}
	ret := []types.UsableInstance{}
	for _, host := range result.Data {
		id, _ := host.GetString("id")
		if len(id) == 0 {
			continue
		}
		name, _ := host.GetString("name")
		machine, err := machines.MachineManager.GetMachineByResourceId(id)
		if err != nil {
			return nil, err
		}
		if machine != nil {
			continue
		}
		ret = append(ret, types.UsableInstance{
			Id:   id,
			Name: name,
			Type: types.MachineResourceTypeBaremetal,
		})
	}
	return ret, nil
}

func (d *SSystemYKEDriver) ValidateCreateData(uctx context.Context, serCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	return yunion_host.ValidateClusterCreateData(data)
}

func (d *SSystemYKEDriver) GetUsableInstances(s *mcclient.ClientSession) ([]types.UsableInstance, error) {
	return GetUsableCloudHosts(s)
}

func (d *SSystemYKEDriver) GetKubeconfig(cluster *clusters.SCluster) (string, error) {
	c, err := models.ClusterManager.FetchClusterByIdOrName(nil, cluster.GetName())
	if err != nil {
		return "", err
	}
	return c.GetAdminKubeconfig()
}

func (d *SSystemYKEDriver) ValidateCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, c *clusters.SCluster, data []*types.CreateMachineData) error {
	if _, _, err := d.sBaseDriver.ValidateCreateMachines(ctx, userCred, c, data); err != nil {
		return err
	}
	return yunion_host.ValidateCreateMachines(data)
}

func (d *SSystemYKEDriver) CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, data []*types.CreateMachineData) ([]manager.IMachine, error) {
	v1Cluster, err := yunion_host.GetV1Cluster(cluster)
	if err != nil {
		return nil, err
	}
	nodesAddData, err := system_yke.GetClusterAddNodesData(cluster, data)
	if err != nil {
		return nil, err
	}

	nodes, err := v1Cluster.AddMachinesToNodes(ctx, userCred, nodesAddData)
	ret := make([]manager.IMachine, 0)
	for _, n := range nodes {
		v2m, err := n.GetV2Machine()
		if err != nil {
			return nil, err
		}
		ret = append(ret, v2m)
	}

	return ret, nil
}

func (d *SSystemYKEDriver) RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	v1Cluster, err := yunion_host.GetV1Cluster(cluster)
	if err != nil {
		return err
	}
	nodes, err := d.GetNodesByMachines(ms)
	if err != nil {
		return err
	}
	return v1Cluster.StartClusterDeployTask(ctx, userCred, models.FetchClusterDeployTaskData(nodes), "")
}

func (d *SSystemYKEDriver) ValidateDeleteCondition() error {
	return nil
}

func (d *SSystemYKEDriver) GetNodeByMachine(machine manager.IMachine) (*models.SNode, error) {
	return models.NodeManager.FetchNodeByHostId(machine.GetResourceId())
}

func (d *SSystemYKEDriver) GetNodesByMachines(machines []manager.IMachine) ([]*models.SNode, error) {
	ret := make([]*models.SNode, 0)
	for _, m := range machines {
		node, err := d.GetNodeByMachine(m)
		if err != nil {
			return nil, err
		}
		ret = append(ret, node)
	}
	return ret, nil
}

func (d *SSystemYKEDriver) getDeleteNodesData(machines []manager.IMachine) (*jsonutils.JSONDict, error) {
	nodes, err := d.GetNodesByMachines(machines)
	if err != nil {
		return nil, err
	}
	nodeObjs := jsonutils.NewArray()
	for _, n := range nodes {
		nodeObjs.Add(jsonutils.NewString(n.GetId()))
	}
	data := jsonutils.NewDict()
	data.Add(nodeObjs, "nodes")
	return data, nil
}

func (d *SSystemYKEDriver) ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machines []manager.IMachine) error {
	//workClusters, err := clusters.ClusterManager.GetNonSystemClusters()
	//if err != nil {
	//return err
	//}
	//if len(workClusters) > 0 {
	//return httperrors.NewNotAcceptableError("%d non system clusters exists, remove them firstly", len(workClusters))
	//}
	oldMachines, err := cluster.GetMachines()
	if err != nil {
		return err
	}
	for _, m := range machines {
		if len(oldMachines) != len(machines) && m.IsFirstNode() {
			return httperrors.NewInputParameterError("First control node %q must deleted at last, old: %d, now: %d", m.GetName(), len(oldMachines), len(machines))
		}
	}
	v1Cluster, err := yunion_host.GetV1Cluster(cluster)
	if err != nil {
		return err
	}
	data, err := d.getDeleteNodesData(machines)
	if err != nil {
		return err
	}
	_, err = v1Cluster.ValidateDeleteNodes(ctx, userCred, data)
	return err
}

func (d *SSystemYKEDriver) RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machines []manager.IMachine, task taskman.ITask) error {
	v1Cluster, err := yunion_host.GetV1Cluster(cluster)
	if err != nil {
		return err
	}
	data, err := d.getDeleteNodesData(machines)
	if err != nil {
		return err
	}
	_, err = v1Cluster.PerformDeleteNodesWithTaskId(ctx, userCred, nil, data, task.GetTaskId())
	return err
}
