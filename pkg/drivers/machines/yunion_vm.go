package machines

import (
	"context"
	"encoding/base64"
	"fmt"

	"yunion.io/x/cluster-api-provider-onecloud/pkg/cloud/onecloud/services/certificates"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/yunion-kube/pkg/drivers/machines/kubeadm"
	"yunion.io/x/yunion-kube/pkg/drivers/machines/userdata"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

type SYunionVMDriver struct {
	*sClusterAPIBaseDriver
}

func NewYunionVMDriver() *SYunionVMDriver {
	return &SYunionVMDriver{
		sClusterAPIBaseDriver: newClusterAPIBaseDriver(),
	}
}

func init() {
	driver := NewYunionVMDriver()
	machines.RegisterMachineDriver(driver)
}

func (d *SYunionVMDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeOnecloud
}

func (d *SYunionVMDriver) GetResourceType() types.MachineResourceType {
	return types.MachineResourceTypeVm
}

func (d *SYunionVMDriver) ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	resType, _ := data.GetString("resource_type")
	if resType != types.MachineResourceTypeVm {
		return httperrors.NewInputParameterError("Invalid resource type: %q", resType)
	}
	resId := jsonutils.GetAnyString(data, []string{"instance", "resource_id"})
	if len(resId) != 0 {
		return httperrors.NewInputParameterError("Resource id must not provide")
	}

	return d.sClusterAPIBaseDriver.ValidateCreateData(session, userCred, ownerProjId, query, data)
}

func (d *SYunionVMDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *machines.SMachine, data *jsonutils.JSONDict) error {
	return d.sClusterAPIBaseDriver.PostCreate(ctx, userCred, cluster, machine, data)
}

func (d *SYunionVMDriver) getServerCreateInput(machine *machines.SMachine, userdata string) (*api.ServerCreateInput, error) {
	userdata = base64.StdEncoding.EncodeToString([]byte(userdata))
	input := new(api.ServerCreateInput)
	input.Name = machine.Name
	input.UserData = userdata
	input.IsSystem = true
	input.VmemSize = 1024
	input.VcpuCount = 2
	input.Disks = []*api.DiskConfig{
		&api.DiskConfig{
			ImageId: "k8s-centos-7.qcow2",
			SizeMb:  102400,
		},
	}
	input.Project = machine.ProjectId
	return input, nil
}

func (d *SYunionVMDriver) getUserData(machine *machines.SMachine, data *machines.MachinePrepareData) (string, error) {
	var userData string
	var err error

	caCertHash, err := certificates.GenerateCertificateHash(data.CAKeyPair.Cert)
	if err != nil {
		return "", err
	}

	cluster, err := machine.GetCluster()
	if err != nil {
		return "", err
	}
	/*baseConfigure := getUserDataBaseConfigure(session, cluster, machine)*/
	apiServerEndpoint, err := cluster.GetAPIServerEndpoint()
	if err != nil {
		return "", err
	}

	switch data.Role {
	case types.RoleTypeControlplane:
		if data.BootstrapToken != "" {
			log.Infof("Allowing a machine to join the control plane")
			updatedJoinConfiguration := kubeadm.SetJoinNodeConfigurationOverrides(caCertHash, data.BootstrapToken, apiServerEndpoint, nil)
			updatedJoinConfiguration = kubeadm.SetControlPlaneJoinConfigurationOverrides(updatedJoinConfiguration)
			joinConfigurationYAML, err := kubeadm.ConfigurationToYAML(updatedJoinConfiguration)
			if err != nil {
				return "", err
			}
			userData, err = userdata.NewJoinControlPlaneCloudInit(&userdata.ControlPlaneJoinInputCloudInit{
				CACert:            string(data.CAKeyPair.Cert),
				CAKey:             string(data.CAKeyPair.Key),
				EtcdCACert:        string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:         string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:  string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:   string(data.FrontProxyCAKeyPair.Key),
				SaCert:            string(data.SAKeyPair.Cert),
				SaKey:             string(data.SAKeyPair.Key),
				JoinConfiguration: joinConfigurationYAML,
			})
			if err != nil {
				return "", err
			}
		} else {
			log.Infof("Machine is the first control plane machine for the cluster")
			if !data.CAKeyPair.HasCertAndKey() {
				return "", fmt.Errorf("failed to run controlplane, missing CAPrivateKey")
			}

			clusterConfiguration, err := kubeadm.SetClusterConfigurationOverrides(cluster, nil)
			if err != nil {
				return "", err
			}
			clusterConfigYAML, err := kubeadm.ConfigurationToYAML(clusterConfiguration)
			if err != nil {
				return "", err
			}

			initConfiguration := kubeadm.SetInitConfigurationOverrides(nil)
			initConfigYAML, err := kubeadm.ConfigurationToYAML(initConfiguration)
			if err != nil {
				return "", err
			}

			userData, err = userdata.NewControlPlaneCloudInit(&userdata.ControlPlaneInputCloudInit{
				CACert:               string(data.CAKeyPair.Cert),
				CAKey:                string(data.CAKeyPair.Key),
				EtcdCACert:           string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:            string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:     string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:      string(data.FrontProxyCAKeyPair.Key),
				SaCert:               string(data.SAKeyPair.Cert),
				SaKey:                string(data.SAKeyPair.Key),
				ClusterConfiguration: clusterConfigYAML,
				InitConfiguration:    initConfigYAML,
			})

			if err != nil {
				return "", err
			}
		}
	case types.RoleTypeNode:
		apiServerEndpoint, err := cluster.GetAPIServerEndpoint()
		if err != nil {
			return "", err
		}
		joinConfiguration := kubeadm.SetJoinNodeConfigurationOverrides(caCertHash, data.BootstrapToken, apiServerEndpoint, nil)
		joinConfigurationYAML, err := kubeadm.ConfigurationToYAML(joinConfiguration)
		if err != nil {
			return "", err
		}
		userData, err = userdata.NewNodeCloudInit(&userdata.NodeInputCloudInit{
			JoinConfiguration: joinConfigurationYAML,
		})
		if err != nil {
			return "", err
		}
	}
	return userData, nil
}

func (d *SYunionVMDriver) PrepareResource(session *mcclient.ClientSession, machine *machines.SMachine, data *machines.MachinePrepareData) (jsonutils.JSONObject, error) {
	// 1. get userdata
	// 2. create vm
	// 3. wait vm running
	// 4. check service
	_, err := machine.GetModelManager().TableSpec().Update(machine, func() error {
		if data.FirstNode {
			machine.FirstNode = tristate.True
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	userdata, err := d.getUserData(machine, data)
	if err != nil {
		return nil, err
	}
	input, err := d.getServerCreateInput(machine, userdata)
	if err != nil {
		return nil, err
	}
	helper := onecloudcli.NewServerHelper(session)
	ret, err := helper.Create(session, input.JSON(input))
	if err != nil {
		return nil, err
	}
	id, err := ret.GetString("id")
	if err != nil {
		return nil, err
	}
	// wait server running and check service
	if err := helper.WaitRunning(id); err != nil {
		return nil, fmt.Errorf("Wait server %d running error: %v", err)
	}
	_, err = helper.ObjectIsExists(id)
	if err != nil {
		return nil, err
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return nil, err
	}
	ip, err := d.GetPrivateIP(session, id)
	if err != nil {
		return nil, err
	}
	_, err = ssh.RemoteSSHCommand(ip, 22, "root", privateKey, "ls -alh /tmp")
	return nil, err
}

func (d *SYunionVMDriver) TerminateResource(session *mcclient.ClientSession, machine *machines.SMachine) error {
	srvId := machine.ResourceId
	if len(srvId) == 0 {
		return nil
	}
	ip, _ := d.GetPrivateIP(session, srvId)
	if len(ip) != 0 {
		return nil
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return err
	}
	_, err = ssh.RemoteSSHCommand(ip, 22, "root", privateKey, "kubeadm reset -f")
	return err
}

func (d *SYunionVMDriver) GetPrivateIP(session *mcclient.ClientSession, id string) (string, error) {
	ret, err := cloudmod.Servernetworks.ListDescendent(session, id, nil)
	if err != nil {
		return "", err
	}
	if len(ret.Data) == 0 {
		return "", fmt.Errorf("Not found networks by id: %s", id)
	}
	return ret.Data[0].GetString("ip_addr")
}
