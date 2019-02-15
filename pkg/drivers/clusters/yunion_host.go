package clusters

import (
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	providerv1 "yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/drivers/clusters/addons"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
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

func (d *SYunoinHostDriver) CreateClusterResource(man *clusters.SClusterManager, data *clusters.ClusterCreateResource) error {
	k8sCli, err := man.GetGlobalK8sClient()
	if err != nil {
		return err
	}
	namespace := data.Namespace
	if err := d.EnsureNamespace(k8sCli, namespace); err != nil {
		return err
	}

	clusterSpec := &providerv1.OneCloudClusterProviderSpec{}
	if data.Vip != "" {
		clusterSpec.NetworkSpec = providerv1.NetworkSpec{
			StaticLB: &providerv1.StaticLB{IPAddress: data.Vip},
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
				Services:      clusterv1.NetworkRanges{[]string{data.ServiceCIDR}},
				Pods:          clusterv1.NetworkRanges{[]string{data.PodCIDR}},
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
