package clusters

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"yunion.io/x/log"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/drivers/clusters/addons"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/options"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
	"yunion.io/x/yunion-kube/pkg/utils/rand"
	"yunion.io/x/yunion-kube/pkg/utils/registry"
	"yunion.io/x/yunion-kube/pkg/utils/ssh"
)

type SYunionVMDriver struct {
	*sClusterAPIDriver
}

func NewYunionVMDriver() *SYunionVMDriver {
	return &SYunionVMDriver{
		sClusterAPIDriver: newClusterAPIDriver(apis.ModeTypeSelfBuild, apis.ProviderTypeOnecloud, apis.ClusterResourceTypeGuest),
	}
}

func init() {
	models.RegisterClusterDriver(NewYunionVMDriver())
}

func (d *SYunionVMDriver) GetMode() apis.ModeType {
	return apis.ModeTypeSelfBuild
}

func (d *SYunionVMDriver) GetProvider() apis.ProviderType {
	return apis.ProviderTypeOnecloud
}

func (d *SYunionVMDriver) GetResourceType() apis.ClusterResourceType {
	return apis.ClusterResourceTypeGuest
}

func (d *SYunionVMDriver) GetK8sVersions() []string {
	return []string{
		"v1.14.1",
	}
}

func getClusterMachineIndexs(cluster *models.SCluster, role string, count int) ([]int, error) {
	if count == 0 {
		return nil, nil
	}
	orderGen := func(count int) []int {
		ret := make([]int, 0)
		for i := 0; i < count; i++ {
			ret = append(ret, i)
		}
		return ret
	}
	if cluster == nil {
		return orderGen(count), nil
	}
	ms, err := cluster.GetMachinesByRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "Get machines by role %s", role)
	}
	idxs := make(map[int]bool)
	for _, m := range ms {
		name := m.GetName()
		parts := strings.Split(name, "-")
		if len(parts) == 0 {
			continue
		}
		idxStr := parts[len(parts)-1]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			log.Errorf("Invalid machine name: %s", name)
			continue
		}
		idxs[idx] = true
	}
	if len(idxs) == 0 {
		return orderGen(count), nil
	}

	ret := make([]int, 0)

	for i := 0; i < count; i++ {
		for idx := 0; ; idx++ {
			_, ok := idxs[idx]
			if !ok {
				ret = append(ret, idx)
				idxs[idx] = true
				break
			}
		}
	}
	return ret, nil
}

func generateVMName(cluster, role, randStr string, idx int) string {
	return fmt.Sprintf("%s-%s-%s-%d", cluster, role, randStr, idx)
}

func (d *SYunionVMDriver) findImage(session *mcclient.ClientSession) (string, error) {
	// TODO: use image tag to find
	//onecloudcli.GetKubernetesImage(session)
	imageName := options.Options.GuestDefaultTemplate
	ret, err := onecloudcli.GetImage(session, imageName)
	if err != nil {
		return "", err
	}
	status, err := ret.GetString("status")
	if err != nil {
		return "", errors.Wrapf(err, "Get image %s status", imageName)
	}
	if status != "active" {
		return "", errors.Errorf("Image %s status is %s", imageName, status)
	}
	return ret.GetString("id")
}

func (d *SYunionVMDriver) ValidateCreateMachines(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *models.SCluster,
	imageRepo *apis.ImageRepository,
	data []*apis.CreateMachineData,
) error {
	controls, nodes, err := d.sClusterAPIDriver.ValidateCreateMachines(ctx, userCred, cluster, data)
	if err != nil {
		return err
	}

	var namePrefix string
	if cluster == nil {
		ret := ctx.Value("VmNamePrefix")
		if ret == nil {
			return errors.New("VmNamePrefix not in context")
		}
		namePrefix = ret.(string)
		imageRepo = models.ClusterManager.GetImageRepository(imageRepo)
	} else {
		namePrefix = cluster.GetName()
		imageRepo, err = cluster.GetImageRepository()
		if err != nil {
			return errors.Wrap(err, "get cluster image repo")
		}
	}

	session, err := models.ClusterManager.GetSession()
	if err != nil {
		return err
	}
	imageId, err := d.findImage(session)
	if err != nil {
		return httperrors.NewInputParameterError("Invalid kubernetes image: %v", err)
	}
	randStr := rand.String(4)
	controlIdxs, err := getClusterMachineIndexs(cluster, apis.RoleTypeControlplane, len(controls))
	if err != nil {
		return httperrors.NewNotAcceptableError("Generate controlplane machines name: %v", err)
	}
	for idx, m := range controls {
		if len(m.Name) == 0 {
			m.Name = generateVMName(namePrefix, m.Role, randStr, controlIdxs[idx])
		}
		if err := d.applyMachineCreateConfig(m, imageId); err != nil {
			return httperrors.NewInputParameterError("Apply controlplane vm config: %v", err)
		}
	}
	nodeIdxs, err := getClusterMachineIndexs(cluster, apis.RoleTypeNode, len(nodes))
	if err != nil {
		return httperrors.NewNotAcceptableError("Generate node machines name: %v", err)
	}
	for idx, m := range nodes {
		if len(m.Name) == 0 {
			m.Name = generateVMName(namePrefix, m.Role, randStr, nodeIdxs[idx])
		}
		if err := d.applyMachineCreateConfig(m, imageId); err != nil {
			return httperrors.NewInputParameterError("Apply node vm config: %v", err)
		}
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return errors.Wrapf(err, "failed to get cloud ssh privateKey")
	}
	var errgrp errgroup.Group
	for _, m := range data {
		tmp := m
		errgrp.Go(func() error {
			if err := d.validateCreateMachine(session, privateKey, tmp); err != nil {
				return err
			}
			return nil
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	return nil
}

func (d *SYunionVMDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	if err := d.sClusterAPIDriver.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	createData := apis.ClusterCreateInput{}
	if err := data.Unmarshal(&createData); err != nil {
		return httperrors.NewInputParameterError("Unmarshal to CreateClusterData: %v", err)
	}
	ms := createData.Machines
	controls, _ := drivers.GetControlplaneMachineDatas("", ms)
	if len(controls) == 0 && createData.Provider != string(apis.ProviderTypeOnecloud) {
		return httperrors.NewInputParameterError("No controlplane nodes")
	}

	ctx = context.WithValue(ctx, "VmNamePrefix", createData.Name)
	imageRepo := createData.ImageRepository
	if err := d.ValidateCreateMachines(ctx, userCred, nil, imageRepo, ms); err != nil {
		return err
	}

	data.Set("machines", jsonutils.Marshal(ms))
	return nil
}

func (d *SYunionVMDriver) applyMachineCreateConfig(m *apis.CreateMachineData, imageId string) error {
	if m.Config == nil {
		m.Config = new(apis.MachineCreateConfig)
	}
	if m.Config.Vm == nil {
		m.Config.Vm = new(apis.MachineCreateVMConfig)
	}
	config := m.Config.Vm
	if config.Hypervisor == "" {
		config.Hypervisor = computeapi.HYPERVISOR_KVM
	}
	if config.VmemSize <= 0 {
		config.VmemSize = apis.DefaultVMMemSize
	}
	if config.VcpuCount <= 0 {
		config.VcpuCount = apis.DefaultVMCPUCount
	}
	if config.VcpuCount < apis.DefaultVMCPUCount {
		return errors.Errorf("cpu count less than %d", apis.DefaultVMCPUCount)
	}
	rootDisk := &computeapi.DiskConfig{
		SizeMb: apis.DefaultVMRootDiskSize,
	}
	restDisks := []*computeapi.DiskConfig{}
	if len(config.Disks) >= 1 {
		rootDisk = config.Disks[0]
		restDisks = config.Disks[1:]
	}
	rootDisk.ImageId = imageId
	config.Disks = []*computeapi.DiskConfig{rootDisk}
	config.Disks = append(config.Disks, restDisks...)
	return nil
}

func (d *SYunionVMDriver) validateCreateMachine(s *mcclient.ClientSession, privateKey string, m *apis.CreateMachineData) error {
	if err := models.ValidateRole(m.Role); err != nil {
		return err
	}
	if m.ResourceType != apis.MachineResourceTypeVm {
		return httperrors.NewInputParameterError("Invalid resource type: %q", m.ResourceType)
	}
	if len(m.ResourceId) != 0 {
		return httperrors.NewInputParameterError("ResourceId can't be specify")
	}
	mDrv := d.GetMachineDriver(apis.MachineResourceType(m.ResourceType))
	if err := mDrv.ValidateCreateData(s, m); err != nil {
		return err
	}
	return nil
}

func (d *SYunionVMDriver) GetUsableInstances(s *mcclient.ClientSession) ([]apis.UsableInstance, error) {
	return nil, httperrors.NewInputParameterError("Can't get UsableInstances")
}

func (d *SYunionVMDriver) GetKubeconfig(cluster *models.SCluster) (string, error) {
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
	helper := onecloudcli.NewServerHelper(session)
	loginInfo, err := helper.GetLoginInfo(masterMachine.GetResourceId())
	if err != nil {
		return "", errors.Wrapf(err, "Get server %q logininfo", masterMachine.GetResourceId())
	}
	if err != nil {
		return "", errors.Wrap(err, "Get server loginInfo")
	}
	out, err := ssh.RemoteSSHCommand(accessIP, 22, loginInfo.Username, loginInfo.Password, privateKey, "cat /etc/kubernetes/admin.conf")
	return out, err
}

func (d *SYunionVMDriver) CreateClusterResource(man *models.SClusterManager, data *apis.ClusterCreateInput) error {
	return d.sClusterAPIDriver.CreateClusterResource(man, data)
}

func (d *SYunionVMDriver) CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, data []*apis.CreateMachineData) ([]manager.IMachine, error) {
	return d.sClusterAPIDriver.CreateMachines(d, ctx, userCred, cluster, data)
}

func (d *SYunionVMDriver) RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	return d.sClusterAPIDriver.RequestDeployMachines(d, ctx, userCred, cluster, ms, task)
}

func (d *SYunionVMDriver) GetAddonsManifest(cluster *models.SCluster) (string, error) {
	commonConf, err := d.GetCommonAddonsConfig(cluster)
	if err != nil {
		return "", err
	}

	reg, err := cluster.GetImageRepository()
	if err != nil {
		return "", err
	}

	pluginConf := &addons.YunionVMPluginsConfig{
		YunionCommonPluginsConfig: commonConf,
		CNICalicoConfig: &addons.CNICalicoConfig{
			ControllerImage: registry.MirrorImage(reg.Url, "kube-controllers", "v3.7.2", "calico"),
			NodeImage:       registry.MirrorImage(reg.Url, "node", "v3.7.2", "calico"),
			CNIImage:        registry.MirrorImage(reg.Url, "cni", "v3.7.2", "calico"),
			ClusterCIDR:     cluster.GetPodCidr(),
		},
	}
	return pluginConf.GenerateYAML()
}
