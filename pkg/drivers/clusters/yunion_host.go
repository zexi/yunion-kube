package clusters

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmconfig "k8s.io/kubernetes/cmd/kubeadm/app/util/config"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	providerv1 "yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers/clusters/addons"
	"yunion.io/x/yunion-kube/pkg/drivers/yunion_host"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/utils/etcd"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

type SYunionHostDriver struct {
	*sClusterAPIDriver
}

func NewYunionHostDriver() *SYunionHostDriver {
	return &SYunionHostDriver{
		sClusterAPIDriver: newClusterAPIDriver(),
	}
}

func init() {
	clusters.RegisterClusterDriver(NewYunionHostDriver())
}

func (d *SYunionHostDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeOnecloud
}

func (d *SYunionHostDriver) GetResourceType() types.ClusterResourceType {
	return types.ClusterResourceTypeHost
}

func (d *SYunionHostDriver) GetK8sVersions() []string {
	return []string{
		"v1.13.3",
	}
}

func (d *SYunionHostDriver) ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	if err := d.sClusterAPIDriver.ValidateCreateData(userCred, ownerProjId, query, data); err != nil {
		return err
	}
	return yunion_host.ValidateClusterCreateData(data)
}

func (d *SYunionHostDriver) GetUsableInstances(s *mcclient.ClientSession) ([]types.UsableInstance, error) {
	return GetUsableCloudHosts(s)
}

func (d *SYunionHostDriver) GetKubeconfig(cluster *clusters.SCluster) (string, error) {
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

func (d *SYunionHostDriver) PreCreateClusterResource(s *mcclient.ClientSession, data *types.CreateClusterData, clusterSpec *providerv1.OneCloudClusterProviderSpec) error {
	if data.HA {
		return nil
	}
	ms := data.Machines
	controls, _ := yunion_host.GetControlplaneMachineDatas(nil, ms)
	if len(controls) == 0 {
		return fmt.Errorf("Empty controlplane machines")
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(s)
	if err != nil {
		return err
	}
	firstControl := controls[0]
	ret, err := yunion_host.ValidateHostId(s, privateKey, firstControl.ResourceId)
	if err != nil {
		return err
	}
	controlIP, err := ret.GetString("access_ip")
	if err != nil {
		return err
	}
	clusterSpec.NetworkSpec = providerv1.NetworkSpec{
		StaticLB: &providerv1.StaticLB{IPAddress: controlIP},
	}
	return nil
}

func (d *SYunionHostDriver) CreateClusterResource(man *clusters.SClusterManager, data *types.CreateClusterData) error {
	k8sCli, err := man.GetGlobalK8sClient()
	if err != nil {
		return err
	}
	namespace := data.Namespace
	if err := d.EnsureNamespace(k8sCli, namespace); err != nil {
		return err
	}

	clusterSpec := &providerv1.OneCloudClusterProviderSpec{}

	if !data.HA {
		controls, _ := yunion_host.GetControlplaneMachineDatas(nil, data.Machines)
		if len(controls) == 0 {
			return fmt.Errorf("Empty controlplane machines")
		}
		firstControl := controls[0]
		session, err := man.GetSession()
		if err != nil {
			return err
		}
		privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
		if err != nil {
			return err
		}
		ret, err := yunion_host.ValidateHostId(session, privateKey, firstControl.ResourceId)
		if err != nil {
			return err
		}
		controlIP, err := ret.GetString("access_ip")
		if err != nil {
			return err
		}
		clusterSpec.NetworkSpec = providerv1.NetworkSpec{
			StaticLB: &providerv1.StaticLB{IPAddress: controlIP},
		}
	}

	providerValue, err := providerv1.EncodeClusterSpec(clusterSpec)
	if err != nil {
		return err
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name: data.Name,
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: clusterv1.ClusterNetworkingConfig{
				Services:      clusterv1.NetworkRanges{[]string{data.ServiceCidr}},
				Pods:          clusterv1.NetworkRanges{[]string{data.PodCidr}},
				ServiceDomain: data.ServiceDomain,
			},
			ProviderSpec: clusterv1.ProviderSpec{
				Value: providerValue,
			},
		},
	}
	cli, err := clusters.ClusterManager.GetGlobalClient()
	if err != nil {
		return httperrors.NewInternalServerError("Get global kubernetes cluster client: %v", err)
	}
	if _, err := cli.ClusterV1alpha1().Clusters(namespace).Create(cluster); err != nil {
		return err
	}
	return nil
}

func (d *SYunionHostDriver) ValidateAddMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, data []*types.CreateMachineData) error {
	if err := yunion_host.ValidateAddMachines(cluster, data); err != nil {
		return err
	}
	return d.sClusterAPIDriver.ValidateAddMachines(ctx, userCred, cluster, data)
}

func (d *SYunionHostDriver) CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, data []*types.CreateMachineData) error {
	return d.sClusterAPIDriver.CreateMachines(d, ctx, userCred, cluster, data)
}

func (d *SYunionHostDriver) RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	return d.sClusterAPIDriver.RequestDeployMachines(d, ctx, userCred, cluster, ms, task)
}

func (d *SYunionHostDriver) ValidateAddMachine(cluster *clusters.SCluster, machine *types.CreateMachineData) error {
	if err := yunion_host.ValidateAddMachines(cluster, []*types.CreateMachineData{machine}); err != nil {
		return err
	}
	cli, err := clusters.ClusterManager.GetGlobalClient()
	if err != nil {
		return httperrors.NewInternalServerError("Get global kubernetes cluster client: %v", err)
	}
	if _, err := cli.ClusterV1alpha1().Machines(machine.Name).Get(machine.Name, v1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return httperrors.NewDuplicateResourceError("Machine %s", machine.Name)
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
		CNIImage:           "registry.cn-beijing.aliyuncs.com/yunionio/cni:v2.7.0",
		CloudProviderImage: "registry.cn-beijing.aliyuncs.com/yunionio/cloud-controller-manager:v2.7.0",
		CSIAttacher:        "registry.cn-beijing.aliyuncs.com/yunionio/csi-attacher:v0.4.0",
		CSIProvisioner:     "registry.cn-beijing.aliyuncs.com/yunionio/csi-provisioner:v0.4.0",
		CSIRegistrar:       "registry.cn-beijing.aliyuncs.com/yunionio/driver-registrar:v0.4.0",
		CSIImage:           "registry.cn-beijing.aliyuncs.com/yunionio/csi-plugin:v2.7.0",
		TillerImage:        "registry.cn-beijing.aliyuncs.com/yunionio/tiller:v2.11.0",
		MetricsServerImage: "registry.cn-beijing.aliyuncs.com/yunionio/metrics-server-amd64:v0.3.1",
	})
}

func (d *SYunionHostDriver) GetClusterEtcdEndpoints(cluster *clusters.SCluster) ([]string, error) {
	ms, err := cluster.GetControlplaneMachines()
	if err != nil {
		return nil, err
	}
	endpoints := []string{}
	for _, m := range ms {
		ip, err := m.GetPrivateIP()
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, etcd.GetClientURLByIP(ip))
	}
	return endpoints, nil
}

func (d *SYunionHostDriver) GetClusterEtcdClient(cluster *clusters.SCluster) (*etcd.Client, error) {
	spec, err := d.GetClusterAPIClusterSpec(cluster)
	if err != nil {
		return nil, err
	}
	ca := string(spec.EtcdCAKeyPair.Cert)
	cert := string(spec.EtcdCAKeyPair.Cert)
	key := string(spec.EtcdCAKeyPair.Key)
	if err != nil {
		return nil, err
	}
	endpoints, err := d.GetClusterEtcdEndpoints(cluster)
	if err != nil {
		return nil, err
	}
	return etcd.New(endpoints, ca, cert, key)
}

func (d *SYunionHostDriver) RemoveEtcdMember(etcdCli *etcd.Client, ip string) error {
	// notifies the other members of the etcd cluster about the removing member
	etcdPeerAddress := etcd.GetPeerURL(ip)

	log.Infof("[etcd] get the member id from peer: %s", etcdPeerAddress)
	id, err := etcdCli.GetMemberID(etcdPeerAddress)
	if err != nil {
		return err
	}

	log.Infof("[etcd] removing etcd member: %s, id: %d", etcdPeerAddress, id)
	members, err := etcdCli.RemoveMember(id)
	if err != nil {
		return err
	}
	log.Infof("[etcd] Updated etcd member list: %v", members)
	return nil
}

func (d *SYunionHostDriver) removeKubeadmClusterStatusAPIEndpoint(status *kubeadmapi.ClusterStatus, m manager.IMachine) error {
	ip, err := m.GetPrivateIP()
	if err != nil {
		return err
	}
	for hostname, endpoint := range status.APIEndpoints {
		if hostname == m.GetName() {
			delete(status.APIEndpoints, hostname)
			return nil
		}
		if endpoint.AdvertiseAddress == ip {
			delete(status.APIEndpoints, hostname)
			return nil
		}
	}
	return nil
}

func (d *SYunionHostDriver) updateKubeadmClusterStatus(cli clientset.Interface, status *kubeadmapi.ClusterStatus) error {
	configMap, err := d.getKubeadmConfigmap(cli)
	if err != nil {
		return err
	}
	clusterStatusYaml, err := kubeadmconfig.MarshalKubeadmConfigObject(status)
	if err != nil {
		return err
	}
	configMap.Data[kubeadmconstants.ClusterStatusConfigMapKey] = string(clusterStatusYaml)
	_, err = cli.CoreV1().ConfigMaps(v1.NamespaceSystem).Update(configMap)
	return err
}

func (d *SYunionHostDriver) RemoveEtcdMembers(cluster *clusters.SCluster, ms []manager.IMachine) error {
	joinControls := make([]manager.IMachine, 0)
	for _, m := range ms {
		if m.IsControlplane() && !m.IsFirstNode() {
			joinControls = append(joinControls, m)
		}
	}
	if len(joinControls) == 0 {
		return nil
	}
	etcdCli, err := d.GetClusterEtcdClient(cluster)
	if err != nil {
		return err
	}
	defer etcdCli.Cleanup()
	clusterStatus, err := d.GetKubeadmClusterStatus(cluster)
	if err != nil {
		return err
	}
	for _, m := range joinControls {
		ip, err := m.GetPrivateIP()
		if err != nil {
			return err
		}
		if err := d.removeKubeadmClusterStatusAPIEndpoint(clusterStatus, m); err != nil {
			return err
		}
		if err := d.RemoveEtcdMember(etcdCli, ip); err != nil {
			if strings.Contains(err.Error(), "not found") {
				continue
			}
			return err
		}
	}
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return err
	}
	if err := d.updateKubeadmClusterStatus(cli, clusterStatus); err != nil {
		return err
	}
	return nil
}

func (d *SYunionHostDriver) RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, ms []manager.IMachine) error {
	if err := d.CleanNodeRecords(cluster, ms); err != nil {
		return err
	}
	if err := d.RemoveEtcdMembers(cluster, ms); err != nil {
		return err
	}
	for _, m := range ms {
		if err := m.(*machines.SMachine).StartMachineDeleteTask(ctx, userCred, nil, ""); err != nil {
			return err
		}
	}
	return nil
}
