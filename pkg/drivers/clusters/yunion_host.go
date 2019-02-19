package clusters

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	providerv1 "yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers/clusters/addons"
	"yunion.io/x/yunion-kube/pkg/drivers/yunion_host"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
	"yunion.io/x/yunion-kube/pkg/options"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

type SYunoinHostDriver struct {
	*sClusterAPIDriver
}

func NewYunionHostDriver() *SYunoinHostDriver {
	return &SYunoinHostDriver{
		sClusterAPIDriver: newClusterAPIDriver(),
	}
}

func init() {
	clusters.RegisterClusterDriver(NewYunionHostDriver())
}

func (d *SYunoinHostDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeOnecloud
}

func (d *SYunoinHostDriver) ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	createData := types.CreateClusterData{}
	if err := data.Unmarshal(&createData); err != nil {
		return httperrors.NewInputParameterError("Unmarshal to CreateClusterData: %v", err)
	}
	ms := createData.Machines
	controls, _ := getControlplaneMachineDatas(nil, ms)
	if len(controls) == 0 {
		return httperrors.NewInputParameterError("No controlplane nodes")
	}
	checkMachines := func(ms []*types.CreateMachineData) error {
		for _, m := range ms {
			if err := machines.ValidateRole(m.Role); err != nil {
				return err
			}
			if err := yunion_host.ValidateResourceType(m.ResourceType); err != nil {
				return err
			}
			if len(m.ResourceId) == 0 {
				return httperrors.NewInputParameterError("ResourceId must provided")
			}
			session, err := clusters.ClusterManager.GetSession()
			if err != nil {
				return err
			}
			if _, err := yunion_host.ValidateHostId(session, m.ResourceId); err != nil {
				return err
			}
		}
		return nil
	}
	if err := checkMachines(ms); err != nil {
		return err
	}
	return nil
}

func (d *SYunoinHostDriver) GetKubeconfig(cluster *clusters.SCluster) (string, error) {
	masterMachine, err := cluster.GetControlplaneMachine()
	if err != nil {
		return "", httperrors.NewInternalServerError("Generate kubeconfig err: %v", err)
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

func (d *SYunoinHostDriver) CreateClusterResource(man *clusters.SClusterManager, data *types.CreateClusterData) error {
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
		controls, _ := getControlplaneMachineDatas(nil, data.Machines)
		if len(controls) == 0 {
			return fmt.Errorf("Empty controlplane machines")
		}
		firstControl := controls[0]
		session, err := man.GetSession()
		if err != nil {
			return err
		}
		ret, err := yunion_host.ValidateHostId(session, firstControl.ResourceId)
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

func getControlplaneMachineDatas(cluster *clusters.SCluster, data []*types.CreateMachineData) ([]*types.CreateMachineData, []*types.CreateMachineData) {
	controls := make([]*types.CreateMachineData, 0)
	nodes := make([]*types.CreateMachineData, 0)
	for _, d := range data {
		if cluster != nil {
			d.ClusterId = cluster.GetId()
		}
		if d.Role == types.RoleTypeControlplane {
			controls = append(controls, d)
		} else {
			nodes = append(nodes, d)
		}
	}
	return controls, nodes
}

func (d *SYunoinHostDriver) RequestCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, data []*types.CreateMachineData, task taskman.ITask) error {
	controls, nodes := getControlplaneMachineDatas(cluster, data)
	if len(controls) == 0 {
		return fmt.Errorf("Empty controlplane machines")
	}

	firstData := controls[0]
	firstMachine, err := machines.MachineManager.CreateMachine(ctx, userCred, firstData)
	if err != nil {
		return err
	}
	// wait first controlplane machine running
	if err := machines.WaitMachineRunning(firstMachine.(*machines.SMachine)); err != nil {
		return fmt.Errorf("Create first controlplane machine error: %v", err)
	}

	// create rest join controlplane
	if len(controls) > 1 {
		for _, d := range controls[1:] {
			_, err = machines.MachineManager.CreateMachine(ctx, userCred, d)
			if err != nil {
				return err
			}
		}
	}

	// create rest nodes
	for _, d := range nodes {
		_, err = machines.MachineManager.CreateMachine(ctx, userCred, d)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *SYunoinHostDriver) ValidateAddMachine(man *clusters.SClusterManager, machine *types.Machine) error {
	cli, err := man.GetGlobalClient()
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

func (d *SYunoinHostDriver) GetAddonsManifest(cluster *clusters.SCluster) (string, error) {
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
		CSIAttacher:        "registry.cn-beijing.aliyuncs.com/yunionio/csi-attacher:v0.4.0",
		CSIProvisioner:     "registry.cn-beijing.aliyuncs.com/yunionio/csi-provisioner:v0.4.0",
		CSIRegistrar:       "registry.cn-beijing.aliyuncs.com/yunionio/driver-registrar:v0.4.0",
		CSIImage:           "registry.cn-beijing.aliyuncs.com/yunionio/csi-plugin:latest",
	})
}

func (d *SYunoinHostDriver) RequestDeleteCluster(c *clusters.SCluster) error {
	cli, err := clusters.ClusterManager.GetGlobalClient()
	if err != nil {
		return httperrors.NewInternalServerError("Get global kubernetes cluster client: %v", err)
	}
	if err := cli.ClusterV1alpha1().Clusters(c.GetNamespace()).Delete(c.Name, &v1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) || strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}
