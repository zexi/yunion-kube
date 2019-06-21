package machines

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"

	"yunion.io/x/cluster-api-provider-onecloud/pkg/cloud/onecloud/services/certificates"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers/machines/kubeadm"
	"yunion.io/x/yunion-kube/pkg/drivers/machines/userdata"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
	"yunion.io/x/yunion-kube/pkg/options"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
	"yunion.io/x/yunion-kube/pkg/utils/registry"
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

func (d *SYunionVMDriver) getServerCreateInput(machine *machines.SMachine, prepareInput *apis.MachinePrepareInput, userdataContent string) (*api.ServerCreateInput, error) {
	var err error
	log.Debugf("Gnerate userdata: %s", userdataContent)
	userdataContent, err = userdata.CompressUserdata(userdataContent)
	if err != nil {
		return nil, errors.Wrap(err, "compress user data")
	}
	tmp := false
	config := prepareInput.Config.Vm
	input := &api.ServerCreateInput{
		ServerConfigs:   new(api.ServerConfigs),
		Name:            machine.Name,
		UserData:        userdataContent,
		IsSystem:        true,
		VmemSize:        config.VmemSize,
		VcpuCount:       config.VcpuCount,
		AutoStart:       true,
		DisableDelete:   &tmp,
		EnableCloudInit: true,
	}
	input.Disks = config.Disks
	input.Networks = config.Networks
	input.IsolatedDevices = config.IsolatedDevices

	input.Project = machine.ProjectId
	input.PreferRegion = config.PreferRegion
	input.PreferZone = config.PreferZone
	input.PreferWire = config.PreferWire
	input.PreferHost = config.PreferHost
	return input, nil
}

func GetDefaultDockerConfig() *apis.DockerConfig {
	o := options.Options
	return &apis.DockerConfig{
		Graph: apis.DefaultDockerGraphDir,
		RegistryMirrors: []string{
			apis.DefaultDockerRegistryMirror1,
			apis.DefaultDockerRegistryMirror2,
			apis.DefaultDockerRegistryMirror3,
		},
		InsecureRegistries: []string{},
		Bip:                o.DockerdBip,
		LiveRestore:        true,
		//ExecOpts:           []string{"native.cgroupdriver=systemd"},
		ExecOpts:  []string{"native.cgroupdriver=cgroupfs"},
		LogDriver: "json-file",
		LogOpts: apis.DockerConfigLogOpts{
			MaxSize: "100m",
		},
		StorageDriver: "overlay2",
	}
}

func (d *SYunionVMDriver) getUserData(machine *machines.SMachine, data *apis.MachinePrepareInput) (string, error) {
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

	kubeletExtraArgs := map[string]string{
		"cgroup-driver":             "cgroupfs",
		"read-only-port":            "10255",
		"pod-infra-container-image": "registry.cn-beijing.aliyuncs.com/yunionio/pause-amd64:3.1",
		"feature-gates":             "CSIPersistentVolume=true,KubeletPluginsWatcher=true,VolumeScheduling=true",
		"eviction-hard":             "memory.available<100Mi,nodefs.available<2Gi,nodefs.inodesFree<5%",
	}
	/*baseConfigure := getUserDataBaseConfigure(session, cluster, machine)*/
	dockerConfig := jsonutils.Marshal(GetDefaultDockerConfig()).PrettyString()
	switch data.Role {
	case types.RoleTypeControlplane:
		if data.BootstrapToken != "" {
			log.Infof("Allowing a machine to join the control plane")
			apiServerEndpoint, err := cluster.GetAPIServerEndpoint()
			if err != nil {
				return "", err
			}
			updatedJoinConfiguration := kubeadm.SetJoinNodeConfigurationOverrides(caCertHash, data.BootstrapToken, apiServerEndpoint, nil)
			updatedJoinConfiguration = kubeadm.SetControlPlaneJoinConfigurationOverrides(updatedJoinConfiguration)
			joinConfigurationYAML, err := kubeadm.ConfigurationToYAML(updatedJoinConfiguration)
			if err != nil {
				return "", err
			}
			userData, err = userdata.NewJoinControlPlaneCloudInit(&userdata.ControlPlaneJoinInputCloudInit{
				DockerConfig:      dockerConfig,
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
				return "", errors.New("failed to run controlplane, missing CAPrivateKey")
			}

			clusterConfiguration, err := kubeadm.SetClusterConfigurationOverrides(cluster, nil)
			if err != nil {
				return "", errors.Wrap(err, "SetClusterConfigurationOverrides")
			}
			clusterConfiguration.APIServer.ExtraArgs = map[string]string{
				"cloud-provider": "external",
				"feature-gates":  "CSIPersistentVolume=true",
				//"runtime-config": "storage.k8s.io/v1alpha1=true,admissionregistration.k8s.io/v1alpha1=true,settings.k8s.io/v1alpha1=true",
			}
			clusterConfiguration.ControllerManager.ExtraArgs = map[string]string{
				"cloud-provider": "external",
				"feature-gates":  "CSIPersistentVolume=true",
			}
			clusterConfiguration.Scheduler.ExtraArgs = map[string]string{
				"feature-gates": "CSIPersistentVolume=true",
			}
			clusterConfiguration.ImageRepository = registry.DefaultRegistryMirror
			clusterConfigYAML, err := kubeadm.ConfigurationToYAML(clusterConfiguration)
			if err != nil {
				return "", errors.Wrap(err, "ConfigurationToYAML")
			}

			initConfiguration := kubeadm.SetInitConfigurationOverrides(&kubeadmv1beta1.InitConfiguration{
				NodeRegistration: kubeadmv1beta1.NodeRegistrationOptions{
					KubeletExtraArgs: kubeletExtraArgs,
				},
			})
			initConfigYAML, err := kubeadm.ConfigurationToYAML(initConfiguration)
			if err != nil {
				return "", err
			}

			kubeProxyConfiguration := kubeadm.SetKubeProxyConfigurationOverrides(nil, cluster.GetServiceCidr())
			kubeProxyConfigYAML, err := kubeadm.KubeProxyConfigurationToYAML(kubeProxyConfiguration)
			if err != nil {
				return "", err
			}

			userData, err = userdata.NewControlPlaneCloudInit(&userdata.ControlPlaneInputCloudInit{
				DockerConfig:           dockerConfig,
				CACert:                 string(data.CAKeyPair.Cert),
				CAKey:                  string(data.CAKeyPair.Key),
				EtcdCACert:             string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:              string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:       string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:        string(data.FrontProxyCAKeyPair.Key),
				SaCert:                 string(data.SAKeyPair.Cert),
				SaKey:                  string(data.SAKeyPair.Key),
				ClusterConfiguration:   clusterConfigYAML,
				InitConfiguration:      initConfigYAML,
				KubeProxyConfiguration: kubeProxyConfigYAML,
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
		joinConfiguration.NodeRegistration.KubeletExtraArgs = kubeletExtraArgs
		joinConfigurationYAML, err := kubeadm.ConfigurationToYAML(joinConfiguration)
		if err != nil {
			return "", err
		}
		userData, err = userdata.NewNodeCloudInit(&userdata.NodeInputCloudInit{
			DockerConfig:      dockerConfig,
			JoinConfiguration: joinConfigurationYAML,
		})
		if err != nil {
			return "", err
		}
	}
	return userData, nil
}

func (d *SYunionVMDriver) GetMachineInitScript(machine *machines.SMachine) (string, error) {
	var script string
	var err error

	switch machine.GetRole() {
	case string(types.RoleTypeNode):
		script = kubeadm.GetNodeJoinScript()
	case string(types.RoleTypeControlplane):
		if machine.IsFirstNode() {
			script = kubeadm.GetControlplaneInitScript()
		} else {
			script = kubeadm.GetControlplaneJoinScript()
		}
	default:
		err = fmt.Errorf("Invalid machine role: %s", machine.GetRole())
	}
	return script, err
}

func (d *SYunionVMDriver) PrepareResource(session *mcclient.ClientSession, machine *machines.SMachine, data *apis.MachinePrepareInput) (jsonutils.JSONObject, error) {
	// 1. get userdata
	// 2. create vm
	// 3. wait vm running
	// 4. check service
	userdata, err := d.getUserData(machine, data)
	if err != nil {
		return nil, errors.Wrap(err, "getUserData")
	}
	input, err := d.getServerCreateInput(machine, data, userdata)
	if err != nil {
		return nil, errors.Wrap(err, "get server create input")
	}
	helper := onecloudcli.NewServerHelper(session)
	ret, err := helper.Create(session, input.JSON(input))
	if err != nil {
		return nil, errors.Wrapf(err, "create server with input: %#v", input)
	}
	id, err := ret.GetString("id")
	if err != nil {
		return nil, err
	}
	machine.SetResourceId(id)
	// wait server running and check service
	if err := helper.WaitRunning(id); err != nil {
		return nil, fmt.Errorf("Wait server %d running error: %v", err)
	}
	_, err = helper.ObjectIsExists(id)
	if err != nil {
		return nil, err
	}
	script, err := d.GetMachineInitScript(machine)
	if err != nil {
		return nil, errors.Wrapf(err, "get machine %s init script", machine.GetName())
	}
	_, err = d.RemoteRunScript(session, id, script)
	return nil, err
}

type ServerLoginInfo struct {
	*onecloudcli.ServerLoginInfo
	Ip         string
	PrivateKey string
}

func (d *SYunionVMDriver) GetServerLoginInfo(s *mcclient.ClientSession, srvId string) (*ServerLoginInfo, error) {
	helper := onecloudcli.NewServerHelper(s)
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(s)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudSSHPrivateKey")
	}
	ip, err := d.GetPrivateIP(s, srvId)
	if err != nil {
		return nil, errors.Wrapf(err, "Get server %q PrivateIP", srvId)
	}
	loginInfo, err := helper.GetLoginInfo(srvId)
	if err != nil {
		return nil, errors.Wrapf(err, "Get server %q loginInfo", srvId)
	}
	return &ServerLoginInfo{
		ServerLoginInfo: loginInfo,
		Ip:              ip,
		PrivateKey:      privateKey,
	}, nil
}

func (d *SYunionVMDriver) RemoteRunScript(s *mcclient.ClientSession, srvId string, script string) (string, error) {
	loginInfo, err := d.GetServerLoginInfo(s, srvId)
	if err != nil {
		return "", errors.Wrap(err, "Get server loginInfo")
	}
	if err := ssh.WaitRemotePortOpen(loginInfo.Ip, 22, 30*time.Second, 10*time.Minute); err != nil {
		return "", errors.Wrapf(err, "remote %s ssh port can't connect", loginInfo.Ip)
	}
	return ssh.RemoteSSHBashScript(loginInfo.Ip, 22, loginInfo.Username, loginInfo.PrivateKey, script)
}

func (d *SYunionVMDriver) RemoteRunCmd(s *mcclient.ClientSession, srvId string, cmd string) (string, error) {
	loginInfo, err := d.GetServerLoginInfo(s, srvId)
	if err != nil {
		return "", errors.Wrap(err, "Get server loginInfo")
	}
	if err := ssh.WaitRemotePortOpen(loginInfo.Ip, 22, 30*time.Second, 10*time.Minute); err != nil {
		return "", errors.Wrapf(err, "remote %s ssh port can't connect", loginInfo.Ip)
	}
	return ssh.RemoteSSHCommand(loginInfo.Ip, 22, loginInfo.Username, loginInfo.PrivateKey, cmd)
}

func (d *SYunionVMDriver) TerminateResource(session *mcclient.ClientSession, machine *machines.SMachine) error {
	srvId := machine.ResourceId
	if len(srvId) == 0 {
		//return errors.Errorf("Machine resource id is empty")
		log.Warningf("Machine resource id is empty, skip clean cloud resource")
		return nil
	}
	if len(machine.Address) != 0 && !machine.IsFirstNode() {
		_, err := d.RemoteRunCmd(session, srvId, "sudo kubeadm reset -f")
		if err != nil {
			return errors.Wrap(err, "kubeadm reset failed")
		}
	}
	helper := onecloudcli.NewServerHelper(session)
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "override_pending_delete")
	_, err := helper.DeleteWithParam(session, srvId, params, nil)
	if err != nil {
		return errors.Wrapf(err, "delete server %s", srvId)
	}
	err = helper.WaitDelete(srvId)
	return err
}

func (d *SYunionVMDriver) GetPrivateIP(session *mcclient.ClientSession, id string) (string, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "system")
	params.Add(jsonutils.JSONTrue, "admin")
	ret, err := cloudmod.Servernetworks.ListDescendent(session, id, params)
	if err != nil {
		return "", err
	}
	if len(ret.Data) == 0 {
		return "", fmt.Errorf("Not found networks by id: %s", id)
	}
	return ret.Data[0].GetString("ip_addr")
}
