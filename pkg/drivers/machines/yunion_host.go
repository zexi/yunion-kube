package machines

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers/yunion_host"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

type SYunionHostDriver struct {
	*sClusterAPIBaseDriver
}

func NewYunionHostDriver() *SYunionHostDriver {
	return &SYunionHostDriver{
		sClusterAPIBaseDriver: newClusterAPIBaseDriver(),
	}
}

func init() {
	driver := NewYunionHostDriver()
	machines.RegisterMachineDriver(driver)
}

func (d *SYunionHostDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeOnecloud
}

func (d *SYunionHostDriver) GetResourceType() types.MachineResourceType {
	return types.MachineResourceTypeBaremetal
}

func (d *SYunionHostDriver) ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	if !userCred.HasSystemAdminPrivilege() {
		return httperrors.NewForbiddenError("Only system admin can use host resource")
	}
	resType, _ := data.GetString("resource_type")
	if err := yunion_host.ValidateResourceType(resType); err != nil {
		return err
	}
	resId := jsonutils.GetAnyString(data, []string{"instance", "resource_id"})
	if len(resId) == 0 {
		return httperrors.NewInputParameterError("Resource id must provide")
	}

	/*role, err := data.GetString("role")
	if err != nil {
		return err
	}
	clusterId, _ := data.GetString("cluster_id")
	if err != nil {
		return err
	}
	controlplaneMachines, err := machines.MachineManager.GetClusterControlplaneMachines(clusterId)
	if err != nil {
		return err
	}
	if role == string(types.RoleTypeControlplane) && len(controlplaneMachines) != 0 {
		return httperrors.NewInputParameterError("Only support one controlplane as for now")
	}*/

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

	data.Set("resource_id", jsonutils.NewString(resId))
	name, _ := ret.Get("name")
	data.Set("name", name)
	return d.sBaseDriver.ValidateCreateData(session, userCred, ownerProjId, query, data)
}

func (d *SYunionHostDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *machines.SMachine, data *jsonutils.JSONDict) error {
	return d.sClusterAPIBaseDriver.PostCreate(ctx, userCred, cluster, machine, data)
}

func createContainerSchedtag(s *mcclient.ClientSession) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(types.ContainerSchedtag), "name")
	params.Add(jsonutils.NewString("Allow run container"), "description")
	_, err := cloudmod.Schedtags.Create(s, params)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate name") {
			return nil
		}
		return err
	}
	return nil
}

func addMachineToContainerSchedtag(s *mcclient.ClientSession, machine *machines.SMachine) error {
	err := createContainerSchedtag(s)
	if err != nil {
		return err
	}
	_, err = cloudmod.Schedtaghosts.Attach(s, types.ContainerSchedtag, machine.ResourceId, nil)
	if err != nil {
		log.Errorf("Add node %s to container schedtag error: %v", machine.Name, err)
	}
	return nil
}

func removeCloudContainers(s *mcclient.ClientSession, machine *machines.SMachine) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(2000), "limit")
	params.Add(jsonutils.JSONTrue, "admin")
	params.Add(jsonutils.NewString("container"), "hypervisor")
	params.Add(jsonutils.NewString(machine.ResourceId), "host")
	result, err := cloudmod.Servers.List(s, params)
	if err != nil {
		return err
	}
	srvIds := []string{}
	for _, srv := range result.Data {
		id, _ := srv.GetString("id")
		srvIds = append(srvIds, id)
	}
	params = jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "override_pending_delete")
	cloudmod.Servers.BatchDeleteWithParam(s, srvIds, params, nil)
	return nil
}

func removeMachineFromContainerSchedtag(s *mcclient.ClientSession, machine *machines.SMachine) error {
	_, err := cloudmod.Schedtaghosts.Detach(s, types.ContainerSchedtag, machine.ResourceId, nil)
	return err
}

func (d *SYunionHostDriver) PrepareResource(session *mcclient.ClientSession, machine *machines.SMachine, data *apis.MachinePrepareInput) (jsonutils.JSONObject, error) {
	hostId := data.InstanceId
	accessIP, err := d.GetPrivateIP(session, hostId)
	if err != nil {
		return nil, err
	}
	data.PrivateIP = accessIP
	cluster, err := machine.GetCluster()
	if err != nil {
		return nil, errors.Wrapf(err, "Get machine %s cluster", machine.GetName())
	}
	apiServerEndpoint, err := cluster.GetAPIServerEndpoint()
	if err != nil {
		return nil, errors.Wrapf(err, "Get cluster %s apiServerEndpoint", cluster.GetName())
	}
	data.ELBAddress = apiServerEndpoint
	_, err = machine.GetModelManager().TableSpec().Update(machine, func() error {
		if data.FirstNode {
			machine.FirstNode = tristate.True
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := addMachineToContainerSchedtag(session, machine); err != nil {
		return nil, err
	}
	userdata, err := d.getUserData(session, machine, data)
	if err != nil {
		return nil, err
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return nil, err
	}
	_, err = ssh.RemoteSSHBashScript(accessIP, 22, "root", privateKey, userdata)
	return nil, err
}

func (d *SYunionHostDriver) ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *machines.SMachine) error {
	return cluster.GetDriver().ValidateDeleteMachines(ctx, userCred, cluster, []manager.IMachine{machine})
}

func (d *SYunionHostDriver) TerminateResource(session *mcclient.ClientSession, machine *machines.SMachine) error {
	hostId := machine.ResourceId
	accessIP, err := d.GetPrivateIP(session, hostId)
	if err != nil {
		return err
	}
	if err := removeCloudContainers(session, machine); err != nil {
		return err
	}
	if err := removeMachineFromContainerSchedtag(session, machine); err != nil {
		log.Errorf("remove machine from container schedtag error: %v", err)
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return err
	}
	_, err = ssh.RemoteSSHCommand(accessIP, 22, "root", privateKey, "kubeadm reset -f")
	return err
}

func (d *SYunionHostDriver) GetPrivateIP(session *mcclient.ClientSession, id string) (string, error) {
	ret, err := cloudmod.Hosts.Get(session, id, nil)
	if err != nil {
		return "", err
	}
	accessIP, _ := ret.GetString("access_ip")
	return accessIP, nil
}
