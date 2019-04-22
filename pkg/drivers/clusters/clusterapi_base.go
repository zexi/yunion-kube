package clusters

import (
	"context"
	"fmt"
	"strings"

	perrors "github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	//kubeadmconfig "k8s.io/kubernetes/cmd/kubeadm/app/util/config"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	providerv1 "yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers/yunion_host"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type sClusterAPIDriver struct {
	*sBaseDriver
}

func newClusterAPIDriver() *sClusterAPIDriver {
	return &sClusterAPIDriver{
		sBaseDriver: newBaseDriver(),
	}
}

func (d *sClusterAPIDriver) UseClusterAPI() bool {
	return true
}

func (d *sClusterAPIDriver) EnsureNamespace(cli *kubernetes.Clientset, namespace string) error {
	ns := apiv1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := cli.CoreV1().Namespaces().Create(&ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (d *sClusterAPIDriver) DeleteNamespace(cli *kubernetes.Clientset, namespace string) error {
	if namespace == apiv1.NamespaceDefault {
		return nil
	}

	err := cli.CoreV1().Namespaces().Delete(namespace, &v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (d *sClusterAPIDriver) PreCreateClusterResource(s *mcclient.ClientSession, data *types.CreateClusterData, clusterSpec *providerv1.OneCloudClusterProviderSpec) error {
	return nil
}

func (d *sClusterAPIDriver) CreateClusterResource(
	drv clusters.IClusterAPIDriver,
	man *clusters.SClusterManager,
	data *types.CreateClusterData,
) error {
	k8sCli, err := man.GetGlobalK8sClient()
	if err != nil {
		return err
	}
	namespace := data.Namespace
	if err := d.EnsureNamespace(k8sCli, namespace); err != nil {
		return err
	}

	session, err := man.GetSession()
	if err != nil {
		return err
	}

	clusterSpec := &providerv1.OneCloudClusterProviderSpec{}

	err = drv.PreCreateClusterResource(session, data, clusterSpec)
	if err != nil {
		return err
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

func (d *sClusterAPIDriver) ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	ok, err := clusters.ClusterManager.IsSystemClusterReady()
	if err != nil {
		return err
	}
	if !ok {
		return httperrors.NewNotAcceptableError("System k8s cluster default not running")
	}
	return nil
}

func (d *sClusterAPIDriver) CreateMachines(
	drv clusters.IClusterDriver,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *clusters.SCluster,
	data []*types.CreateMachineData,
) error {
	needControlplane, err := yunion_host.NeedControlplane(cluster)
	if err != nil {
		return err
	}
	controls, nodes := yunion_host.GetControlplaneMachineDatas(cluster, data)
	if needControlplane {
		if len(controls) == 0 {
			return fmt.Errorf("Empty controlplane machines")
		}
	}
	//cms, nms, err := createMachines(ctx, userCred, controls, nodes)
	_, _, err = createMachines(ctx, userCred, controls, nodes)
	if err != nil {
		return err
	}
	return nil
}

type machineData struct {
	machine *machines.SMachine
	data    *jsonutils.JSONDict
}

func newMachineData(machine *machines.SMachine, input *types.CreateMachineData) *machineData {
	return &machineData{
		machine: machine,
		data:    jsonutils.Marshal(input).(*jsonutils.JSONDict),
	}
}

func createMachines(ctx context.Context, userCred mcclient.TokenCredential, controls, nodes []*types.CreateMachineData) ([]*machineData, []*machineData, error) {
	cms := make([]*machineData, 0)
	nms := make([]*machineData, 0)
	cf := func(data []*types.CreateMachineData) ([]*machineData, error) {
		ret := make([]*machineData, 0)
		for _, m := range data {
			obj, err := machines.MachineManager.CreateMachineNoHook(ctx, userCred, m)
			if err != nil {
				return nil, err
			}
			ret = append(ret, newMachineData(obj.(*machines.SMachine), m))
		}
		return ret, nil
	}
	var err error
	cms, err = cf(controls)
	if err != nil {
		return nil, nil, err
	}
	nms, err = cf(nodes)
	if err != nil {
		return nil, nil, err
	}
	return cms, nms, nil
}

func machinesPostCreate(ctx context.Context, userCred mcclient.TokenCredential, ms []*machineData) {
	for _, m := range ms {
		func() {
			lockman.LockObject(ctx, m.machine)
			defer lockman.ReleaseObject(ctx, m.machine)
			m.machine.PostCreate(ctx, userCred, userCred.GetTenantId(), nil, m.data)
		}()
	}
}

type IClusterAPIDriver interface {
	clusters.IClusterDriver
}

func (d *sClusterAPIDriver) RequestDeployMachines(
	drv clusters.IClusterDriver,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *clusters.SCluster,
	ms []manager.IMachine,
	task taskman.ITask,
) error {
	var firstCm *machines.SMachine
	var restMachines []*machines.SMachine
	var needControlplane bool

	doPostCreate := func(m *machines.SMachine) {
		lockman.LockObject(ctx, m)
		defer lockman.ReleaseObject(ctx, m)
		m.PostCreate(ctx, userCred, userCred.GetTenantId(), nil, jsonutils.NewDict())
	}

	for _, m := range ms {
		if m.IsFirstNode() {
			firstCm = m.(*machines.SMachine)
			needControlplane = true
		} else {
			restMachines = append(restMachines, m.(*machines.SMachine))
		}
	}

	if needControlplane {
		// TODO: fix this
		masterIP, err := firstCm.GetPrivateIP()
		if err != nil {
			log.Errorf("Get privateIP error: %v", err)
		}
		if len(masterIP) != 0 {
			if err := d.updateClusterStaticLBAddress(cluster, masterIP); err != nil {
				return err
			}
		}
		doPostCreate(firstCm)
		// wait first controlplane machine running
		if err := machines.WaitMachineRunning(firstCm); err != nil {
			return fmt.Errorf("Create first controlplane machine error: %v", err)
		}
	}

	// create rest join controlplane
	for _, d := range restMachines {
		doPostCreate(d)
	}

	return nil
}

func (d *sClusterAPIDriver) UpdateClusterResource(c *clusters.SCluster, spec *providerv1.OneCloudClusterProviderSpec) (*clusterv1.Cluster, error) {
	cli, err := clusters.ClusterManager.GetGlobalClient()
	if err != nil {
		return nil, err
	}
	obj, err := d.GetClusterAPICluster(c)
	if err != nil {
		return nil, err
	}
	providerValue, err := providerv1.EncodeClusterSpec(spec)
	if err != nil {
		return nil, err
	}
	obj.Spec.ProviderSpec.Value = providerValue
	return cli.ClusterV1alpha1().Clusters(c.GetNamespace()).Update(obj)
}

func (d *sClusterAPIDriver) updateClusterStaticLBAddress(c *clusters.SCluster, ip string) error {
	clusterSpec := &providerv1.OneCloudClusterProviderSpec{}
	clusterSpec.NetworkSpec = providerv1.NetworkSpec{
		StaticLB: &providerv1.StaticLB{IPAddress: ip},
	}
	_, err := d.UpdateClusterResource(c, clusterSpec)
	return err
}

func (d *sClusterAPIDriver) ValidateAddMachine(c *clusters.SCluster, machine *types.CreateMachineData) error {
	needControlplane, err := yunion_host.NeedControlplane(c)
	if err != nil {
		return err
	}
	if needControlplane && machine.Role != types.RoleTypeControlplane {
		return httperrors.NewInputParameterError("controlplane node must created")
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

func (d *sClusterAPIDriver) GetAddonsManifest(cluster *clusters.SCluster) (string, error) {
	return "", nil
}

func (d *sClusterAPIDriver) RequestDeleteCluster(c *clusters.SCluster) error {
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

func (d *sClusterAPIDriver) ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, ms []manager.IMachine) error {
	oldMachines, err := cluster.GetMachines()
	if err != nil {
		return err
	}
	for _, m := range ms {
		if len(oldMachines) != len(ms) && m.IsFirstNode() {
			return httperrors.NewInputParameterError("First control node %q must deleted at last", m.GetName())
		}
	}
	return nil
}

func (d *sClusterAPIDriver) GetClusterAPICluster(cluster *clusters.SCluster) (*clusterv1.Cluster, error) {
	cli, err := clusters.ClusterManager.GetGlobalClient()
	if err != nil {
		return nil, err
	}
	obj, err := cli.ClusterV1alpha1().Clusters(cluster.GetNamespace()).Get(cluster.GetName(), v1.GetOptions{})
	return obj, err
}

func (d *sClusterAPIDriver) GetClusterAPIClusterSpec(cluster *clusters.SCluster) (*providerv1.OneCloudClusterProviderSpec, error) {
	c, err := d.GetClusterAPICluster(cluster)
	if err != nil {
		return nil, err
	}
	return providerv1.ClusterConfigFromProviderSpec(c.Spec.ProviderSpec)
}

func (d *sClusterAPIDriver) CleanNodeRecords(clusters *clusters.SCluster, ms []manager.IMachine) error {
	deleteNodes := make([]manager.IMachine, 0)
	for _, m := range ms {
		if !m.IsFirstNode() {
			deleteNodes = append(deleteNodes, m)
		}
	}
	cli, err := clusters.GetK8sClient()
	if err != nil {
		return err
	}
	for _, n := range deleteNodes {
		cli.CoreV1().Nodes().Delete(n.GetName(), &v1.DeleteOptions{})
	}
	return nil
}

func (d *sClusterAPIDriver) getKubeadmConfigmap(cli kubernetes.Interface) (*apiv1.ConfigMap, error) {
	configMap, err := cli.CoreV1().ConfigMaps(v1.NamespaceSystem).Get(kubeadmconstants.KubeadmConfigConfigMap, v1.GetOptions{})
	if err != nil {
		return nil, perrors.Wrap(err, "failed to get config map")
	}
	return configMap, nil
}

func (d *sClusterAPIDriver) GetKubeadmClusterStatus(cluster *clusters.SCluster) (*kubeadmapi.ClusterStatus, error) {
	log.Infof("Reading clusterstatus from cluster: %s", cluster.GetName())
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return nil, err
	}
	configMap, err := d.getKubeadmConfigmap(cli)
	if err != nil {
		return nil, err
	}
	return d.unmarshalClusterStatus(configMap.Data)
}

func (d *sClusterAPIDriver) unmarshalClusterStatus(data map[string]string) (*kubeadmapi.ClusterStatus, error) {
	clusterStatusData, ok := data[kubeadmconstants.ClusterStatusConfigMapKey]
	if !ok {
		return nil, perrors.Errorf("unexpected error when reading kubeadm-config ConfigMap: %s key value pair missing", kubeadmconstants.ClusterStatusConfigMapKey)
	}
	clusterStatus := &kubeadmapi.ClusterStatus{}
	if err := runtime.DecodeInto(kubeadmscheme.Codecs.UniversalDecoder(), []byte(clusterStatusData), clusterStatus); err != nil {
		return nil, err
	}
	return clusterStatus, nil
}

func (d *sClusterAPIDriver) RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, ms []manager.IMachine) error {
	if err := d.CleanNodeRecords(cluster, ms); err != nil {
		return err
	}
	for _, m := range ms {
		if err := m.(*machines.SMachine).StartMachineDeleteTask(ctx, userCred, nil, ""); err != nil {
			return err
		}
	}
	return nil
}
