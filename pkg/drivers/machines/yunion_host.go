package machines

import (
	"fmt"
	"strings"

	"yunion.io/x/cluster-api-provider-onecloud/pkg/cloud/onecloud/services/certificates"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/drivers/machines/userdata"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
	"yunion.io/x/yunion-kube/pkg/options"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

const (
	HostTypeKVM     = "hypervisor"
	HostTypeKubelet = "kubelet"
)

type SYunionHostDriver struct {
	*sBaseDriver
}

func NewYunionHostDriver() *SYunionHostDriver {
	return &SYunionHostDriver{
		sBaseDriver: newBaseDriver(),
	}
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

	role, err := data.GetString("role")
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
	if m := machines.MachineManager.GetMachineByResourceId(resId); m != nil {
		return httperrors.NewInputParameterError("Machine %s already use host %s", m.GetName(), resId)
	}

	data.Set("resource_id", jsonutils.NewString(resId))
	name, _ := ret.Get("name")
	data.Set("name", name)
	return d.sBaseDriver.ValidateCreateData(session, userCred, ownerProjId, query, data)
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
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return nil, err
	}
	_, err = ssh.RemoteSSHBashScript(accessIP, 22, "root", privateKey, userdata)
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
