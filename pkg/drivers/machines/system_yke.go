package machines

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers/system_yke"
	"yunion.io/x/yunion-kube/pkg/drivers/yunion_host"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
)

type SSystemYKEDriver struct {
	*sBaseDriver
}

func NewSystemYKEDriver() *SSystemYKEDriver {
	return &SSystemYKEDriver{
		sBaseDriver: newBaseDriver(),
	}
}

func (d *SSystemYKEDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeSystem
}

func (d *SSystemYKEDriver) GetResourceType() types.MachineResourceType {
	return types.MachineResourceTypeBaremetal
}

func (d *SSystemYKEDriver) ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	clusterId, _ := data.GetString("cluster_id")
	cluster, err := clusters.ClusterManager.GetCluster(clusterId)
	if err != nil {
		return err
	}
	v1Cluster, err := yunion_host.GetV1Cluster(cluster)
	if err != nil {
		return err
	}
	if models.ClusterProcessingStatus.Has(v1Cluster.Status) {
		return httperrors.NewNotAcceptableError(fmt.Sprintf("cluster status is %s", v1Cluster.Status))
	}

	role, err := data.GetString("role")
	if err != nil {
		return err
	}
	v1Roles := []string{}
	if role == types.RoleTypeControlplane {
		v1Roles = append(v1Roles, "etcd", "controlplane")
	}
	if role == types.RoleTypeNode {
		v1Roles = append(v1Roles, "worker")
	}
	rolesObj := jsonutils.Marshal(v1Roles)
	data.Add(rolesObj, "roles")
	resId := jsonutils.GetAnyString(data, []string{"instance", "resource_id"})
	if len(resId) == 0 {
		return httperrors.NewInputParameterError("Resource id must provide")
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return err
	}
	ret, err := yunion_host.ValidateHostId(session, privateKey, resId)
	if err != nil {
		return err
	}
	resId, err = ret.GetString("id")
	if err != nil {
		return err
	}
	name, err := ret.GetString("name")
	if err != nil {
		return err
	}
	data.Add(jsonutils.NewString(resId), "host_id")
	data.Add(jsonutils.NewString(name), "name")
	return nil
}

func (d *SSystemYKEDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *machines.SMachine, data *jsonutils.JSONDict) error {
	v1Cluster, err := yunion_host.GetV1Cluster(cluster)
	if err != nil {
		return err
	}
	node, _ := yunion_host.GetV1Node(machine)
	if node != nil {
		// node already exists
		return nil
	}
	data, err = system_yke.GetClusterAddNodesData(cluster, []*types.CreateMachineData{
		&types.CreateMachineData{
			Name:       machine.Name,
			Role:       machine.Role,
			ResourceId: machine.ResourceId,
		},
	})
	if err != nil {
		return err
	}
	_, err = v1Cluster.PerformAddNodes(ctx, userCred, nil, data)
	return err
}

func (d *SSystemYKEDriver) ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *machines.SMachine) error {
	return httperrors.NewUnsupportOperationError("Global system cluster machine can't be delete directly, call cluster delete-machines")
}

func init() {
	driver := &SSystemYKEDriver{}
	machines.RegisterMachineDriver(driver)
}
