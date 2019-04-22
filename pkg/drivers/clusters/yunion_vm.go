package clusters

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	providerv1 "yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers/yunion_host"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

type SYunionVMDriver struct {
	*sClusterAPIDriver
}

func NewYunionVMDriver() *SYunionVMDriver {
	return &SYunionVMDriver{
		sClusterAPIDriver: newClusterAPIDriver(),
	}
}

func init() {
	clusters.RegisterClusterDriver(NewYunionVMDriver())
}

func (d *SYunionVMDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeOnecloud
}

func (d *SYunionVMDriver) GetResourceType() types.ClusterResourceType {
	return types.ClusterResourceTypeGuest
}

func (d *SYunionVMDriver) GetK8sVersions() []string {
	return []string{
		"v1.14.1",
	}
}

func generateVMName(cluster, role string, idx int) string {
	return fmt.Sprintf("%s-%s-aabb-%d", cluster, role, idx)
}

func (d *SYunionVMDriver) ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	log.Errorf("---Start ValidateCreateData: %v", data.String())
	if err := d.sClusterAPIDriver.ValidateCreateData(userCred, ownerProjId, query, data); err != nil {
		return err
	}
	createData := types.CreateClusterData{}
	if err := data.Unmarshal(&createData); err != nil {
		return httperrors.NewInputParameterError("Unmarshal to CreateClusterData: %v", err)
	}
	ms := createData.Machines
	controls, nodes := yunion_host.GetControlplaneMachineDatas(nil, ms)
	if len(controls) == 0 && createData.Provider != string(types.ProviderTypeOnecloud) {
		return httperrors.NewInputParameterError("No controlplane nodes")
	}
	for idx, m := range controls {
		if len(m.Name) == 0 {
			m.Name = generateVMName(createData.Name, m.Role, idx)
		}
	}
	for idx, m := range nodes {
		if len(m.Name) == 0 {
			m.Name = generateVMName(createData.Name, m.Role, idx)
		}
	}
	session, err := clusters.ClusterManager.GetSession()
	if err != nil {
		return err
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return err
	}
	var errgrp errgroup.Group
	for _, m := range ms {
		tmp := m
		errgrp.Go(func() error {
			if err := validateCreateMachine(session, privateKey, tmp); err != nil {
				return err
			}
			return nil
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	data.Set("machines", jsonutils.Marshal(ms))
	log.Errorf("---end ValidateCreateData: %v", data.String())
	return nil
}

func validateCreateMachine(s *mcclient.ClientSession, privateKey string, m *types.CreateMachineData) error {
	if err := machines.ValidateRole(m.Role); err != nil {
		return err
	}
	if m.ResourceType != types.MachineResourceTypeVm {
		return httperrors.NewInputParameterError("Invalid resource type: %q", m.ResourceType)
	}
	if len(m.ResourceId) != 0 {
		return httperrors.NewInputParameterError("ResourceId can't be specify")
	}
	return nil
}

func (d *SYunionVMDriver) GetUsableInstances(s *mcclient.ClientSession) ([]types.UsableInstance, error) {
	return nil, httperrors.NewInputParameterError("Can't get UsableInstances")
}

func (d *SYunionVMDriver) GetKubeconfig(cluster *clusters.SCluster) (string, error) {
	masterMachine, err := cluster.GetRunningControlplaneMachine()
	if err != nil {
		return "", err
	}
	accessIP, err := masterMachine.GetPrivateIP()
	if err != nil {
		return "", err
	}
	session, err := models.GetAdminSession()
	if err != nil {
		return "", err
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return "", err
	}
	out, err := ssh.RemoteSSHCommand(accessIP, 22, "root", privateKey, "cat /etc/kubernetes/admin.conf")
	return out, err
}

func (d *SYunionVMDriver) UpdateClusterResource(c *clusters.SCluster, spec *providerv1.OneCloudClusterProviderSpec) (*clusterv1.Cluster, error) {
	return d.sClusterAPIDriver.UpdateClusterResource(c, spec)
}

func (d *SYunionVMDriver) CreateClusterResource(man *clusters.SClusterManager, data *types.CreateClusterData) error {
	return d.sClusterAPIDriver.CreateClusterResource(d, man, data)
}

func (d *SYunionVMDriver) CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, data []*types.CreateMachineData) error {
	return d.sClusterAPIDriver.CreateMachines(d, ctx, userCred, cluster, data)
}

func (d *SYunionVMDriver) RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	return d.sClusterAPIDriver.RequestDeployMachines(d, ctx, userCred, cluster, ms, task)
}
