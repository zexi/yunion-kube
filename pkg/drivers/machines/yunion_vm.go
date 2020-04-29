package machines

import (
	"context"
	"fmt"
	"time"
	"yunion.io/x/yunion-kube/pkg/models"

	"github.com/pkg/errors"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers/machines/kubeadm"
	"yunion.io/x/yunion-kube/pkg/drivers/machines/userdata"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/utils/certificates"
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
	models.RegisterMachineDriver(driver)
}

func (d *SYunionVMDriver) GetProvider() apis.ProviderType {
	return apis.ProviderTypeOnecloud
}

func (d *SYunionVMDriver) GetResourceType() apis.MachineResourceType {
	return apis.MachineResourceTypeVm
}

func (d *SYunionVMDriver) ValidateCreateData(session *mcclient.ClientSession, input *apis.CreateMachineData) error {
	if input.ResourceType != apis.MachineResourceTypeVm {
		return httperrors.NewInputParameterError("Invalid resource type: %q", input.ResourceType)
	}
	if len(input.ResourceId) != 0 {
		return httperrors.NewInputParameterError("Resource id must not provide")
	}

	return d.validateConfig(session, input.Config.Vm)
}

func (d *SYunionVMDriver) validateConfig(s *mcclient.ClientSession, config *apis.MachineCreateVMConfig) error {
	if config.VcpuCount < 4 {
		return httperrors.NewNotAcceptableError("CPU count must large than 4")
	}
	if config.VmemSize < 4096 {
		return httperrors.NewNotAcceptableError("Memory size must large than 4G")
	}
	input := &api.ServerCreateInput{
		ServerConfigs: &api.ServerConfigs{
			PreferRegion:     config.PreferRegion,
			PreferZone:       config.PreferZone,
			PreferWire:       config.PreferWire,
			PreferHost:       config.PreferHost,
			PreferBackupHost: config.PreferBackupHost,
			Hypervisor:       config.Hypervisor,
			Disks:            config.Disks,
			Networks:         config.Networks,
			IsolatedDevices:  config.IsolatedDevices,
		},
		VmemSize:  config.VmemSize,
		VcpuCount: config.VcpuCount,
	}
	validateData := input.JSON(input)
	ret, err := cloudmod.Servers.PerformClassAction(s, "check-create-data", validateData)
	log.Infof("check server create data: %s, ret: %s err: %v", validateData, ret, err)
	if err != nil {
		return err
	}
	return nil
}

func (d *SYunionVMDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machine *models.SMachine, data *jsonutils.JSONDict) error {
	return d.sClusterAPIBaseDriver.PostCreate(ctx, userCred, cluster, machine, data)
}

func (d *SYunionVMDriver) getServerCreateInput(machine *models.SMachine, prepareInput *apis.MachinePrepareInput) (*api.ServerCreateInput, error) {
	tmpFalse := false
	tmpTrue := true
	config := prepareInput.Config.Vm
	input := &api.ServerCreateInput{
		ServerConfigs: new(api.ServerConfigs),
		VmemSize:      config.VmemSize,
		VcpuCount:     config.VcpuCount,
		AutoStart:     true,
		//EnableCloudInit: true,
	}
	input.Name = machine.Name
	input.IsSystem = &tmpTrue
	input.DisableDelete = &tmpFalse
	input.Hypervisor = config.Hypervisor
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

func GetDefaultDockerConfig(input *apis.DockerConfig) *apis.DockerConfig {
	o := options.Options
	if input.Graph == "" {
		input.Graph = apis.DefaultDockerGraphDir
	}
	if len(input.RegistryMirrors) == 0 {
		input.RegistryMirrors = []string{
			apis.DefaultDockerRegistryMirror1,
			apis.DefaultDockerRegistryMirror2,
			apis.DefaultDockerRegistryMirror3,
		}
	}
	if len(input.Bip) == 0 {
		input.Bip = o.DockerdBip
	}
	input.LiveRestore = true
	if len(input.ExecOpts) == 0 {
		//ExecOpts:           []string{"native.cgroupdriver=systemd"},
		input.ExecOpts = []string{"native.cgroupdriver=cgroupfs"}
	}
	if input.LogDriver == "" {
		input.LogDriver = "json-file"
		input.LogOpts = apis.DockerConfigLogOpts{
			MaxSize: "100m",
		}
	}
	if input.StorageDriver == "" {
		input.StorageDriver = "overlay2"
	}
	return input
}

func (d *SYunionVMDriver) GetMachineInitScript(machine *models.SMachine, data *apis.MachinePrepareInput, ip string) (string, error) {
	var initScript string
	var err error

	caCertHash, err := certificates.GenerateCertificateHash(data.CAKeyPair.Cert)
	if err != nil {
		return "", err
	}

	cluster, err := machine.GetCluster()
	if err != nil {
		return "", err
	}

	imageRepo := data.Config.ImageRepository
	kubeletExtraArgs := map[string]string{
		"cgroup-driver":             "cgroupfs",
		"read-only-port":            "10255",
		"pod-infra-container-image": fmt.Sprintf("%s/pause-amd64:3.1", imageRepo.Url),
		"feature-gates":             "CSIPersistentVolume=true,KubeletPluginsWatcher=true,VolumeScheduling=true",
		"eviction-hard":             "memory.available<100Mi,nodefs.available<2Gi,nodefs.inodesFree<5%",
	}
	dockerConfig := GetDefaultDockerConfig(data.Config.DockerConfig)
	switch data.Role {
	case apis.RoleTypeControlplane:
		if data.BootstrapToken != "" {
			log.Infof("Allowing a machine to join the control plane")
			apiServerEndpoint, err := cluster.GetAPIServerEndpoint()
			if err != nil {
				return "", err
			}
			updatedJoinConfiguration := kubeadm.SetJoinNodeConfigurationOverrides(caCertHash, data.BootstrapToken, apiServerEndpoint, nil, machine.Name)
			updatedJoinConfiguration = kubeadm.SetControlPlaneJoinConfigurationOverrides(updatedJoinConfiguration, ip)
			initScript, err = userdata.JoinControlplaneConfig{
				DockerConfiguration: dockerConfig,
				CACert:              string(data.CAKeyPair.Cert),
				CAKey:               string(data.CAKeyPair.Key),
				EtcdCACert:          string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:           string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:    string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:     string(data.FrontProxyCAKeyPair.Key),
				SaCert:              string(data.SAKeyPair.Cert),
				SaKey:               string(data.SAKeyPair.Key),
				JoinConfiguration:   updatedJoinConfiguration,
			}.ToScript()
			if err != nil {
				return "", errors.Wrap(err, "generate join controlplane script")
			}
		} else {
			log.Infof("Machine is the first control plane machine for the cluster")
			if !data.CAKeyPair.HasCertAndKey() {
				return "", errors.New("failed to run controlplane, missing CAPrivateKey")
			}

			clusterConfiguration, err := kubeadm.SetClusterConfigurationOverrides(cluster, nil, ip)
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
			clusterConfiguration.ImageRepository = imageRepo.Url

			initConfiguration := kubeadm.SetInitConfigurationOverrides(&kubeadmv1beta1.InitConfiguration{
				NodeRegistration: kubeadmv1beta1.NodeRegistrationOptions{
					KubeletExtraArgs: kubeletExtraArgs,
				},
			}, machine.Name)

			kubeProxyConfiguration := kubeadm.SetKubeProxyConfigurationOverrides(nil, cluster.GetServiceCidr())

			initScript, err = userdata.InitNodeConfig{
				DockerConfiguration:    dockerConfig,
				CACert:                 string(data.CAKeyPair.Cert),
				CAKey:                  string(data.CAKeyPair.Key),
				EtcdCACert:             string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:              string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:       string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:        string(data.FrontProxyCAKeyPair.Key),
				SaCert:                 string(data.SAKeyPair.Cert),
				SaKey:                  string(data.SAKeyPair.Key),
				ClusterConfiguration:   clusterConfiguration,
				InitConfiguration:      initConfiguration,
				KubeProxyConfiguration: kubeProxyConfiguration,
			}.ToScript()

			if err != nil {
				return "", err
			}
		}
	case apis.RoleTypeNode:
		apiServerEndpoint, err := cluster.GetAPIServerEndpoint()
		if err != nil {
			return "", err
		}
		joinConfiguration := kubeadm.SetJoinNodeConfigurationOverrides(caCertHash, data.BootstrapToken, apiServerEndpoint, nil, machine.Name)
		joinConfiguration.NodeRegistration.KubeletExtraArgs = kubeletExtraArgs
		initScript, err = userdata.JoinNodeConfig{
			DockerConfiguration: dockerConfig,
			JoinConfiguration:   joinConfiguration,
		}.ToScript()
		if err != nil {
			return "", err
		}
	}
	return initScript, nil
}

func (d *SYunionVMDriver) PrepareResource(
	session *mcclient.ClientSession,
	machine *models.SMachine,
	data *apis.MachinePrepareInput) (jsonutils.JSONObject, error) {
	// 1. create vm
	// 2. wait vm running
	// 3. ssh run init script
	// 4. check service
	input, err := d.getServerCreateInput(machine, data)
	if err != nil {
		return nil, errors.Wrap(err, "get server create input")
	}
	helper := onecloudcli.NewServerHelper(session)
	ret, err := helper.Create(session, input.JSON(input))
	if err != nil {
		log.Errorf("Create server error: %v, input disks: %#v", err, input.Disks[0])
		return nil, errors.Wrapf(err, "create server with input: %#v", input)
	}
	id, err := ret.GetString("id")
	if err != nil {
		return nil, err
	}
	machine.SetHypervisor(input.Hypervisor)
	machine.SetResourceId(id)
	// wait server running and check service
	if err := helper.WaitRunning(id); err != nil {
		return nil, fmt.Errorf("Wait server %d running error: %v", err)
	}
	_, err = helper.ObjectIsExists(id)
	if err != nil {
		return nil, err
	}
	ip, err := d.GetPrivateIP(session, id)
	if err != nil {
		return nil, err
	}
	script, err := d.GetMachineInitScript(machine, data, ip)
	if err != nil {
		return nil, errors.Wrapf(err, "get machine %s init script", machine.GetName())
	}
	log.Debugf("Generate script: %s", script)
	output, err := d.RemoteRunScript(session, id, script)
	if err != nil {
		return nil, errors.Wrapf(err, "output: %s", output)
	}
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
	return ssh.RemoteSSHBashScript(loginInfo.Ip, 22, loginInfo.Username, loginInfo.Password, loginInfo.PrivateKey, script)
}

func (d *SYunionVMDriver) RemoteRunCmd(s *mcclient.ClientSession, srvId string, cmd string) (string, error) {
	loginInfo, err := d.GetServerLoginInfo(s, srvId)
	if err != nil {
		return "", errors.Wrap(err, "Get server loginInfo")
	}
	if err := ssh.WaitRemotePortOpen(loginInfo.Ip, 22, 30*time.Second, 10*time.Minute); err != nil {
		return "", errors.Wrapf(err, "remote %s ssh port can't connect", loginInfo.Ip)
	}
	return ssh.RemoteSSHCommand(loginInfo.Ip, 22, loginInfo.Username, loginInfo.Password, loginInfo.PrivateKey, cmd)
}

func (d *SYunionVMDriver) TerminateResource(session *mcclient.ClientSession, machine *models.SMachine) error {
	srvId := machine.ResourceId
	if len(srvId) == 0 {
		//return errors.Errorf("Machine resource id is empty")
		log.Warningf("Machine resource id is empty, skip clean cloud resource")
		return nil
	}
	if len(machine.Address) != 0 && !machine.IsFirstNode() {
		_, err := d.RemoteRunCmd(session, srvId, "sudo kubeadm reset -f")
		if err != nil {
			//return errors.Wrap(err, "kubeadm reset failed")
			log.Errorf("kubeadm reset failed: %v", err)
		}
	}
	helper := onecloudcli.NewServerHelper(session)
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "override_pending_delete")
	_, err := helper.DeleteWithParam(session, srvId, params, nil)
	if err != nil {
		if onecloudcli.IsNotFoundError(err) {
			return nil
		}
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
