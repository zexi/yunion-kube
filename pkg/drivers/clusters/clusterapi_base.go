package clusters

import (
	"context"
	"fmt"

	perrors "github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	//kubeadmconfig "k8s.io/kubernetes/cmd/kubeadm/app/util/config"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/drivers/clusters/addons"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/utils/registry"
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

func (d *sClusterAPIDriver) NeedGenerateCertificate() bool {
	return true
}

func (d *sClusterAPIDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	/*ok, err := clusters.ClusterManager.IsSystemClusterReady()
	if err != nil {
		return err
	}
	if !ok {
		return httperrors.NewNotAcceptableError("System k8s cluster default not running")
	}*/
	return nil
}

func (d *sClusterAPIDriver) CreateMachines(
	drv clusters.IClusterDriver,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *clusters.SCluster,
	data []*types.CreateMachineData,
) ([]manager.IMachine, error) {
	needControlplane, err := cluster.NeedControlplane()
	if err != nil {
		return nil, err
	}
	controls, nodes := drivers.GetControlplaneMachineDatas(cluster.GetId(), data)
	if needControlplane {
		if len(controls) == 0 {
			return nil, fmt.Errorf("Empty controlplane machines")
		}
	}
	cms, nms, err := createMachines(ctx, userCred, controls, nodes)
	if err != nil {
		return nil, err
	}
	ret := make([]manager.IMachine, 0)
	for _, m := range cms {
		ret = append(ret, m.machine)
	}
	for _, m := range nms {
		ret = append(ret, m.machine)
	}
	return ret, nil
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
		//masterIP, err := firstCm.GetPrivateIP()
		//if err != nil {
		//log.Errorf("Get privateIP error: %v", err)
		//}
		//if len(masterIP) != 0 {
		//if err := d.updateClusterStaticLBAddress(cluster, masterIP); err != nil {
		//return err
		//}
		//}
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
	task.ScheduleRun(nil)
	return nil
}

func (d *sClusterAPIDriver) GetAddonsManifest(cluster *clusters.SCluster) (string, error) {
	return "", nil
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

func (d *sClusterAPIDriver) RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	items := make([]db.IStandaloneModel, 0)
	for _, m := range ms {
		items = append(items, m.(db.IStandaloneModel))
	}
	return machines.MachineManager.StartMachineBatchDeleteTask(ctx, userCred, items, nil, task.GetTaskId())
}

func (d *sClusterAPIDriver) GetAddonYunionAuthConfig(cluster *clusters.SCluster) addons.YunionAuthConfig {
	o := options.Options
	authConfig := addons.YunionAuthConfig{
		AuthUrl:       o.AuthURL,
		AdminUser:     o.AdminUser,
		AdminPassword: o.AdminPassword,
		AdminProject:  o.AdminProject,
		Region:        o.Region,
		Cluster:       cluster.GetName(),
		InstanceType:  cluster.ResourceType,
	}
	return authConfig
}

func (d *sClusterAPIDriver) GetCommonAddonsConfig(cluster *clusters.SCluster) *addons.YunionCommonPluginsConfig {
	authConfig := d.GetAddonYunionAuthConfig(cluster)

	commonConf := &addons.YunionCommonPluginsConfig{
		MetricsPluginConfig: &addons.MetricsPluginConfig{
			MetricsServerImage: registry.MirrorImage("metrics-server-amd64", "v0.3.1", ""),
		},
		HelmPluginConfig: &addons.HelmPluginConfig{
			TillerImage: registry.MirrorImage("tiller", "v2.11.0", ""),
		},
		CloudProviderYunionConfig: &addons.CloudProviderYunionConfig{
			YunionAuthConfig:   authConfig,
			CloudProviderImage: registry.MirrorImage("yunion-cloud-controller-manager", "v2.9.0", ""),
		},
		CSIYunionConfig: &addons.CSIYunionConfig{
			YunionAuthConfig: authConfig,
			AttacherImage:    registry.MirrorImage("csi-attacher", "v1.0.1", ""),
			ProvisionerImage: registry.MirrorImage("csi-provisioner", "v1.0.1", ""),
			RegistrarImage:   registry.MirrorImage("csi-node-driver-registrar", "v1.1.0", ""),
			PluginImage:      registry.MirrorImage("yunion-csi-plugin", "v2.9.0", ""),
			Base64Config:     authConfig.ToJSONBase64String(),
		},
		IngressControllerYunionConfig: &addons.IngressControllerYunionConfig{
			YunionAuthConfig: authConfig,
			Image:            registry.MirrorImage("yunion-ingress-controller", "v2.10.0", ""),
		},
	}

	return commonConf
}