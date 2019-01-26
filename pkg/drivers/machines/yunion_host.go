package machines

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	tcmd "k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clientcmd"
	"sigs.k8s.io/cluster-api/pkg/util"

	"yunion.io/x/cluster-api-provider-onecloud/pkg/cloud/onecloud/services/certificates"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/drivers/machines/addons"
	"yunion.io/x/yunion-kube/pkg/drivers/machines/userdata"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

const (
	HostTypeKVM     = "hypervisor"
	HostTypeKubelet = "kubelet"
)

const (
	retryIntervalKubectlApply = 10 * time.Second
	timeoutKubectlApply       = 15 * time.Minute
)

type SYunionHostDriver struct {
}

func init() {
	driver := &SYunionHostDriver{}
	machines.RegisterMachineDriver(driver)
}

func (d *SYunionHostDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeOnecloud
}

func (d *SYunionHostDriver) ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	if !userCred.HasSystemAdminPrivelege() {
		return httperrors.NewForbiddenError("Only system admin can use host resource")
	}
	resType, _ := data.GetString("resource_type")
	if resType != types.MachineResourceTypeBaremetal {
		return httperrors.NewInputParameterError("Invalid resource type: %q", resType)
	}
	resId := jsonutils.GetAnyString(data, []string{"instance", "resource_id"})
	if len(resId) == 0 {
		return httperrors.NewInputParameterError("Resource id must provide")
	}
	ret, err := cloudmod.Hosts.Get(session, resId, nil)
	if err != nil {
		return err
	}
	hostType, _ := ret.GetString("host_type")
	if !utils.IsInStringArray(hostType, []string{HostTypeKVM, HostTypeKubelet}) {
		return httperrors.NewInputParameterError("Host %q invalid host_type %q", resId, hostType)
	}
	resId, _ = ret.GetString("id")
	data.Set("resource_id", jsonutils.NewString(resId))
	name, _ := ret.Get("name")
	data.Set("name", name)
	return nil
}

func getUserDataBaseConfigure(session *mcclient.ClientSession, cluster *clusters.SCluster, machine *machines.SMachine) userdata.BaseConfigure {
	o := options.Options
	schedulerUrl, err := session.GetServiceURL("scheduler", "internalURL")
	if err != nil {
		log.Errorf("Get internal scheduler endpoint error: %v", err)
	}
	return userdata.BaseConfigure{
		DockerConfigure: userdata.DockerConfigure{},
		OnecloudConfigure: userdata.OnecloudConfigure{
			AuthURL:           o.AuthURL,
			AdminUser:         o.AdminUser,
			AdminPassword:     o.AdminPassword,
			AdminProject:      o.AdminProject,
			Region:            o.Region,
			Cluster:           cluster.Name,
			SchedulerEndpoint: schedulerUrl,
		},
	}
}

func (d *SYunionHostDriver) getUserData(session *mcclient.ClientSession, machine *machines.SMachine, data *machines.MachinePrepareData) (string, error) {
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

	baseConfigure := getUserDataBaseConfigure(session, cluster, machine)

	// apply userdata values based on the role of the machine
	switch data.Role {
	case types.RoleTypeControlplane:
		if data.BootstrapToken != "" {
			log.Infof("Allow machine %q to join control plane for cluster %q", machine.Name, cluster.Name)
			userData, err = userdata.JoinControlPlane(&userdata.ControlPlaneJoinInput{
				BaseConfigure:    baseConfigure,
				CACert:           string(data.CAKeyPair.Cert),
				CAKey:            string(data.CAKeyPair.Key),
				CACertHash:       caCertHash,
				EtcdCACert:       string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:        string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert: string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:  string(data.FrontProxyCAKeyPair.Key),
				SaCert:           string(data.SAKeyPair.Cert),
				SaKey:            string(data.SAKeyPair.Key),
				BootstrapToken:   data.BootstrapToken,
				ELBAddress:       data.ELBAddress,
				PrivateIP:        data.PrivateIP,
			})
			if err != nil {
				return "", err
			}
		} else {
			log.Infof("Machine %q is the first controlplane machine for cluster %q", machine.Name, cluster.Name)
			if !data.CAKeyPair.HasCertAndKey() {
				return "", fmt.Errorf("failed to run controlplane, missing CAPrivateKey")
			}

			userData, err = userdata.NewControlPlane(&userdata.ControlPlaneInput{
				BaseConfigure:     baseConfigure,
				CACert:            string(data.CAKeyPair.Cert),
				CAKey:             string(data.CAKeyPair.Key),
				EtcdCACert:        string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:         string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:  string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:   string(data.FrontProxyCAKeyPair.Key),
				SaCert:            string(data.SAKeyPair.Cert),
				SaKey:             string(data.SAKeyPair.Key),
				ELBAddress:        data.ELBAddress,
				PrivateIP:         data.PrivateIP,
				ClusterName:       cluster.Name,
				PodSubnet:         cluster.PodCidr,
				ServiceSubnet:     cluster.ServiceCidr,
				ServiceDomain:     cluster.ServiceDomain,
				KubernetesVersion: cluster.Version,
			})
			if err != nil {
				return "", err
			}
		}
	case types.RoleTypeNode:
		userData, err = userdata.NewNode(&userdata.NodeInput{
			BaseConfigure:  baseConfigure,
			CACertHash:     caCertHash,
			BootstrapToken: data.BootstrapToken,
			ELBAddress:     data.ELBAddress,
		})
		if err != nil {
			return "", err
		}
	}
	return userData, nil
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

func (d *SYunionHostDriver) PrepareResource(session *mcclient.ClientSession, machine *machines.SMachine, data *machines.MachinePrepareData) (jsonutils.JSONObject, error) {
	hostId := data.InstanceId
	accessIP, err := d.GetPrivateIP(session, hostId)
	if err != nil {
		return nil, err
	}
	data.PrivateIP = accessIP
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
	_, err = ssh.RemoteSSHBashScript("root", accessIP, "123@openmag", userdata)
	return nil, err
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
	_, err = ssh.RemoteSSHBashScript("root", accessIP, "123@openmag", "'kubeadm reset -f'")
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

func (d *SYunionHostDriver) GetKubeConfig(session *mcclient.ClientSession, machine *machines.SMachine) (string, error) {
	hostId := machine.ResourceId
	accessIP, err := d.GetPrivateIP(session, hostId)
	if err != nil {
		return "", err
	}
	out, err := ssh.RemoteSSHBashScript("root", accessIP, "123@openmag", "'cat /etc/kubernetes/admin.conf'")
	return out, err
}

func (d *SYunionHostDriver) GetAddonsManifest(cluster *clusters.SCluster) (string, error) {
	o := options.Options
	return addons.GetYunionManifest(addons.ManifestConfig{
		ClusterCIDR:        cluster.ServiceCidr,
		AuthURL:            o.AuthURL,
		AdminUser:          o.AdminUser,
		AdminPassword:      o.AdminPassword,
		AdminProject:       o.AdminProject,
		Region:             o.Region,
		KubeCluster:        cluster.Name,
		CNIImage:           "registry.cn-beijing.aliyuncs.com/yunionio/cni:latest",
		CloudProviderImage: "registry.cn-beijing.aliyuncs.com/yunionio/cloud-controller-manager:latest",
	})
}

func (d *SYunionHostDriver) ApplyAddons(cluster *clusters.SCluster, kubeconfig string) error {
	cli, err := NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		return err
	}
	manifest, err := d.GetAddonsManifest(cluster)
	if err != nil {
		return err
	}
	return cli.Apply(manifest)
}

type client struct {
	kubeconfigFile  string
	configOverrides tcmd.ConfigOverrides
	closeFn         func() error
}

func NewClientFromKubeconfig(kubeconfig string) (*client, error) {
	f, err := createTempFile(kubeconfig)
	if err != nil {
		return nil, err
	}
	defer ifErrRemove(err, f)
	//c, err := NewFromDefaultSearchPath(f, clientcmd.NewConfigOverrides())
	//if err != nil {
	//return nil, err
	//}
	c := &client{
		kubeconfigFile:  f,
		configOverrides: clientcmd.NewConfigOverrides(),
	}
	c.closeFn = c.removeKubeconfigFile
	return c, nil
}

func createTempFile(contents string) (string, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	defer ifErrRemove(err, f.Name())
	if err = f.Close(); err != nil {
		return "", err
	}
	err = ioutil.WriteFile(f.Name(), []byte(contents), 0644)
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

func ifErrRemove(err error, path string) {
	if err != nil {
		if err := os.Remove(path); err != nil {
			log.Warningf("Error removing file '%s': %v", path, err)
		}
	}
}

func (c *client) removeKubeconfigFile() error {
	return os.Remove(c.kubeconfigFile)
}

func (c *client) kubectlManifestCmd(commandName, manifest string) error {
	cmd := exec.Command("kubectl", c.buildKubectlArgs(commandName)...)
	cmd.Stdin = strings.NewReader(manifest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("couldn't kubectl apply, output: %s, error: %v", string(output), err)
	}
	return nil
}

func (c *client) buildKubectlArgs(commandName string) []string {
	args := []string{commandName}
	if c.kubeconfigFile != "" {
		args = append(args, "--kubeconfig", c.kubeconfigFile)
	}
	if c.configOverrides.Context.Cluster != "" {
		args = append(args, "--cluster", c.configOverrides.Context.Cluster)
	}
	if c.configOverrides.Context.Namespace != "" {
		args = append(args, "--namespace", c.configOverrides.Context.Namespace)
	}
	if c.configOverrides.Context.AuthInfo != "" {
		args = append(args, "--user", c.configOverrides.Context.AuthInfo)
	}
	return append(args, "-f", "-")
}

func (c *client) Apply(manifest string) error {
	return c.waitForKubectlApply(manifest)
}

func (c *client) kubectlDelete(manifest string) error {
	return c.kubectlManifestCmd("delete", manifest)
}

func (c *client) kubectlApply(manifest string) error {
	return c.kubectlManifestCmd("apply", manifest)
}

func (c *client) waitForKubectlApply(manifest string) error {
	err := util.PollImmediate(retryIntervalKubectlApply, timeoutKubectlApply, func() (bool, error) {
		log.Infof("Waiting for kubectl apply...")
		err := c.kubectlApply(manifest)
		if err != nil {
			if strings.Contains(err.Error(), "refused") {
				// Connection was refused, probably because the API server is not ready yet.
				log.Infof("aiting for kubectl apply... server not yet available: %v", err)
				return false, nil
			}
			if strings.Contains(err.Error(), "unable to recognize") {
				log.Infof("Waiting for kubectl apply... api not yet available: %v", err)
				return false, nil
			}
			log.Warningf("Waiting for kubectl apply... unknown error %v", err)
			return false, err
		}

		return true, nil
	})
	return err
}
