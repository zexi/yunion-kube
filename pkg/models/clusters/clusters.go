package clusters

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/k8s"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	k8sutil "yunion.io/x/yunion-kube/pkg/k8s/util"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

var ClusterManager *SClusterManager

func init() {
	ClusterManager = &SClusterManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SCluster{},
			"kubeclusters_tbl",
			"kubecluster",
			"kubeclusters",
		),
	}
	manager.RegisterClusterManager(ClusterManager)
}

type SClusterManager struct {
	db.SVirtualResourceBaseManager
}

type SCluster struct {
	db.SVirtualResourceBase

	ClusterType   string            `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	CloudType     string            `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Mode          string            `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Provider      string            `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ServiceCidr   string            `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ServiceDomain string            `width:"128" charset:"ascii" nullable:"false" create:"required" list:"user"`
	PodCidr       string            `width:"36" charset:"ascii" nullable:"true" create:"optional" list:"user"`
	Version       string            `width:"128" charset:"ascii" nullable:"false" create:"optional" list:"user"`
	Namespace     string            `nullable:"true" create:"optional" list:"user"`
	Ha            tristate.TriState `nullable:"true" create:"required" list:"user"`
	Kubeconfig    string            `nullable:"true" create:"optional"`
	IsPublic      bool              `default:"false" nullable:"false" index:"true" create:"admin_optional" list:"user" update:"user"`
}

func SetJSONDataDefault(data *jsonutils.JSONDict, key string, defVal string) string {
	val, _ := data.GetString(key)
	if len(val) == 0 {
		val = defVal
		data.Set(key, jsonutils.NewString(val))
	}
	return val
}

func (m *SClusterManager) GetSession() (*mcclient.ClientSession, error) {
	return models.GetAdminSession()
}

func (m *SClusterManager) CreateCluster(ctx context.Context, userCred mcclient.TokenCredential, data types.CreateClusterData) (manager.ICluster, error) {
	input := jsonutils.Marshal(data)
	obj, err := db.DoCreate(m, ctx, userCred, nil, input, userCred.GetTenantId())
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (m *SClusterManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("tenant_id"), owner), sqlchemy.IsTrue(q.Field("is_public"))))
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("pending_deleted")), sqlchemy.IsFalse(q.Field("pending_deleted"))))
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	return q
}

func (m *SClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var (
		clusterType  string
		cloudType    string
		modeType     string
		providerType string
	)

	clusterType = SetJSONDataDefault(data, "cluster_type", string(types.ClusterTypeDefault))
	if !utils.IsInStringArray(clusterType, []string{string(types.ClusterTypeDefault)}) {
		return nil, httperrors.NewInputParameterError("Invalid cluster type: %q", clusterType)
	}

	cloudType = SetJSONDataDefault(data, "cloud_type", string(types.CloudTypePrivate))
	if !utils.IsInStringArray(cloudType, []string{string(types.CloudTypePrivate)}) {
		return nil, httperrors.NewInputParameterError("Invalid cloud type: %q", cloudType)
	}

	modeType = SetJSONDataDefault(data, "mode", string(types.ModeTypeSelfBuild))
	if !utils.IsInStringArray(modeType, []string{
		string(types.ModeTypeSelfBuild),
		string(types.ModeTypeImport),
	}) {
		return nil, httperrors.NewInputParameterError("Invalid mode type: %q", modeType)
	}

	providerType = SetJSONDataDefault(data, "provider", string(types.ProviderTypeOnecloud))
	if err := ValidateProviderType(providerType); err != nil {
		return nil, err
	}

	serviceCidr := SetJSONDataDefault(data, "service_cidr", types.DefaultServiceCIDR)
	if _, err := netutils.NewIPV4Prefix(serviceCidr); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid service CIDR: %q", serviceCidr)
	}

	serviceDomain := SetJSONDataDefault(data, "service_domain", types.DefaultServiceDomain)
	if len(serviceDomain) == 0 {
		return nil, httperrors.NewInputParameterError("service domain must provided")
	}

	driver := GetDriver(types.ProviderType(providerType))
	if err := driver.ValidateCreateData(userCred, ownerId, query, data); err != nil {
		return nil, err
	}

	versions := driver.GetK8sVersions()
	defaultVersion := versions[0]
	version := SetJSONDataDefault(data, "version", defaultVersion)
	if !utils.IsInStringArray(version, versions) {
		return nil, httperrors.NewInputParameterError("Invalid version: %q, choose one from %v", version, versions)
	}

	if jsonutils.QueryBoolean(data, "ha", false) {
		data.Set("ha", jsonutils.JSONTrue)
	} else {
		data.Set("ha", jsonutils.JSONFalse)
	}

	res := types.CreateClusterData{}
	if err := data.Unmarshal(&res); err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal: %v", err)
	}

	if res.Provider != string(types.ProviderTypeSystem) && len(res.Machines) == 0 {
		return nil, httperrors.NewInputParameterError("Machines desc not provider")
	}

	// TODO: support namespace by userCred??
	res.Namespace = res.Name

	if err := driver.CreateClusterResource(m, &res); err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (m *SClusterManager) AllowGetPropertyK8sVersions(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func ValidateProviderType(providerType string) error {
	if !utils.IsInStringArray(providerType, []string{
		string(types.ProviderTypeOnecloud),
		string(types.ProviderTypeSystem),
	}) {
		return httperrors.NewInputParameterError("Invalid provider type: %q", providerType)
	}
	return nil
}

func (m *SClusterManager) GetPropertyK8sVersions(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	providerType, _ := query.GetString("provider")
	if err := ValidateProviderType(providerType); err != nil {
		return nil, err
	}
	driver := GetDriver(types.ProviderType(providerType))
	versions := driver.GetK8sVersions()
	ret := jsonutils.Marshal(versions)
	return ret, nil
}

func (m *SClusterManager) AllowPerformCheckSystemReady(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return true
}

func (m *SClusterManager) PerformCheckSystemReady(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isReady, err := m.IsSystemClusterReady()
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(isReady), nil
}

func (m *SClusterManager) IsSystemClusterReady() (bool, error) {
	systemCluster, err := m.GetV1SystemCluster()
	if err != nil {
		return false, err
	}
	if systemCluster.ApiEndpoint == "" {
		return false, nil
	}
	if systemCluster.Status == types.ClusterStatusInit {
		return false, nil
	}
	if utils.IsInStringArray(systemCluster.Status, []string{types.ClusterStatusCreating, types.ClusterStatusDeleting}) {
		return false, httperrors.NewNotAcceptableError("System cluster is %s", systemCluster.Status)
	}
	//if systemCluster.Status != types.ClusterStatusRunning {
	//return false, httperrors.NewNotAcceptableError("System cluster status is %s", systemCluster.Status)
	//}
	_, err = systemCluster.GetK8sClient()
	if err != nil {
		return false, httperrors.NewNotAcceptableError("Can't create k8s client to system cluster: %v", err)
	}
	//info, err := cli.Discovery().ServerVersion()
	return true, nil
}

func (m *SClusterManager) AllowGetPropertyUsableInstances(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), m.KeywordPlural(), policy.PolicyActionGet, "usable-instances")
}

func (m *SClusterManager) GetPropertyUsableInstances(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	providerType, _ := query.GetString("provider")
	if err := ValidateProviderType(providerType); err != nil {
		return nil, err
	}
	driver := GetDriver(types.ProviderType(providerType))
	session, err := m.GetSession()
	if err != nil {
		return nil, err
	}
	instances, err := driver.GetUsableInstances(session)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.Marshal(instances)
	return ret, nil
}

func (m *SClusterManager) IsClusterExists(userCred mcclient.TokenCredential, id string) (manager.ICluster, bool, error) {
	obj, err := m.FetchByIdOrName(userCred, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	return obj.(*SCluster), true, nil
}

func (m *SClusterManager) GetNonSystemClusters() ([]manager.ICluster, error) {
	clusters := m.Query().SubQuery()
	q := clusters.Query().Filter(sqlchemy.NotEquals(clusters.Field("provider"), string(types.ProviderTypeSystem)))
	objs := make([]SCluster, 0)
	err := db.FetchModelObjects(m, q, &objs)
	if err != nil {
		return nil, err
	}
	ret := make([]manager.ICluster, len(objs))
	for i := range objs {
		ret[i] = &objs[i]
	}
	return ret, nil
}

func (m *SClusterManager) FetchClusterByIdOrName(userCred mcclient.TokenCredential, id string) (manager.ICluster, error) {
	cluster, err := m.FetchByIdOrName(userCred, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewNotFoundError("Cluster %s", id)
		}
		return nil, err
	}
	return cluster.(*SCluster), nil
}

func (m *SClusterManager) GetGlobalClientConfig() (*rest.Config, error) {
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(nil, "default")
	if err != nil {
		return nil, err
	}
	return cluster.GetK8sRestConfig()
}

func (m *SClusterManager) GetGlobalK8sClient() (*kubernetes.Clientset, error) {
	config, err := m.GetGlobalClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (m *SClusterManager) GetGlobalClient() (*clientset.Clientset, error) {
	conf, err := m.GetGlobalClientConfig()
	if err != nil {
		return nil, err
	}
	return clientset.NewForConfig(conf)
}

func (m *SClusterManager) GetV1SystemCluster() (*models.SCluster, error) {
	return models.ClusterManager.FetchClusterByIdOrName(nil, types.DefaultCluster)
}

func (m *SClusterManager) GetSystemCluster() (*SCluster, error) {
	obj, err := m.FetchByIdOrName(nil, types.DefaultCluster)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (m *SClusterManager) GetCluster(id string) (*SCluster, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (c *SCluster) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.IsOwner(userCred) || c.IsShared() || db.IsAdminAllowGet(userCred, c)
}

func (c *SCluster) IsShared() bool {
	return c.IsPublic
}

func (c *SCluster) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return c.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, c, "public")
}

func (c *SCluster) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return c.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, c, "private")
}

func (c *SCluster) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !c.IsPublic {
		_, err := c.GetModelManager().TableSpec().Update(c, func() error {
			c.IsPublic = true
			return nil
		})
		return nil, err
	}
	return nil, nil
}

func (c *SCluster) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if c.IsPublic {
		_, err := c.GetModelManager().TableSpec().Update(c, func() error {
			c.IsPublic = false
			return nil
		})
		return nil, err
	}
	return nil, nil
}

func (c *SCluster) GetDriver() IClusterDriver {
	return GetDriver(types.ProviderType(c.Provider))
}

func (c *SCluster) GetMachinesCount() (int, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return 0, err
	}
	return len(ms), nil
}

func (c *SCluster) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := c.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return c.moreExtraInfo(extra)
}

func (c *SCluster) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := c.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}

	return c.moreExtraInfo(extra), nil
}

func (c *SCluster) moreExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	if cnt, err := c.GetMachinesCount(); err != nil {
		log.Errorf("GetMachines error: %v", err)
	} else {
		extra.Add(jsonutils.NewInt(int64(cnt)), "machines")
	}
	return extra
}

func (c *SCluster) ValidateAddMachine(machine *types.CreateMachineData) error {
	if !utils.IsInStringArray(c.Status, []string{types.ClusterStatusInit, types.ClusterStatusCreating, types.ClusterStatusRunning}) {
		return httperrors.NewNotAcceptableError("Can't add machine when cluster status is %s", c.Status)
	}
	driver := c.GetDriver()
	return driver.ValidateAddMachine(c, machine)
}

func (c *SCluster) GetNamespace() string {
	if c.Namespace == "" {
		return c.Name
	}
	return c.Namespace
}

func (c *SCluster) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	if err := c.StartClusterCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartClusterCreateTask error: %v", err)
	}
}

func (c *SCluster) StartClusterCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	c.SetStatus(userCred, types.ClusterStatusCreating, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterCreateTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) GetPVCCount() (int, error) {
	cli, err := c.GetK8sClient()
	if err != nil {
		return 0, err
	}
	pvcs, err := k8sutil.GetPVCList(cli, "")
	if err != nil {
		return 0, err
	}
	return len(pvcs.Items), nil
}

func (c *SCluster) CheckPVCEmpty() error {
	pvcCnt, _ := c.GetPVCCount()
	if pvcCnt > 0 {
		return httperrors.NewNotAcceptableError("Cluster has %d PersistentVolumeClaims, clean them firstly", pvcCnt)
	}
	return nil
}

func (c *SCluster) ValidateDeleteCondition(ctx context.Context) error {
	if err := c.GetDriver().ValidateDeleteCondition(); err != nil {
		return err
	}
	if err := c.CheckPVCEmpty(); err != nil {
		return err
	}
	return nil
}

func (c *SCluster) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Cluster delete do nothing")
	return nil
}

func (c *SCluster) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return c.SVirtualResourceBase.Delete(ctx, userCred)
}

func (c *SCluster) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return c.StartClusterDeleteTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (c *SCluster) StartClusterDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := c.SetStatus(userCred, types.ClusterStatusDeleting, ""); err != nil {
		return err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeleteTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) allowPerformAction(userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.IsOwner(userCred)
}

func (c *SCluster) AllowPerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, data)
}

func (c *SCluster) PerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := c.ValidateDeleteCondition(ctx); err != nil {
		return nil, err
	}
	return nil, c.RealDelete(ctx, userCred)
}

func (c *SCluster) AllowGetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, nil)
}

func (c *SCluster) GetRunningControlplaneMachine() (manager.IMachine, error) {
	return c.getControlplaneMachine(true)
}

func (c *SCluster) GetStatus() string {
	return c.Status
}

func (c *SCluster) getControlplaneMachine(checkStatus bool) (manager.IMachine, error) {
	machines, err := c.GetMachines()
	if err != nil {
		return nil, err
	}
	if machines == nil {
		return nil, nil
	}
	for _, m := range machines {
		if m.IsControlplane() && m.IsFirstNode() {
			if !checkStatus {
				return m, nil
			}
			if m.IsRunning() {
				return m, nil
			} else {
				return nil, fmt.Errorf("Not found a running controlplane machine, status is %s", m.GetStatus())
			}
		}
	}
	return nil, fmt.Errorf("Not found a controlplane machine")
}

func (c *SCluster) GetControlplaneMachines() ([]manager.IMachine, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return nil, err
	}
	ret := make([]manager.IMachine, 0)
	for _, m := range ms {
		if m.IsControlplane() {
			ret = append(ret, m)
		}
	}
	return ret, nil
}

func (c *SCluster) GetMachines() ([]manager.IMachine, error) {
	return manager.MachineManager().GetMachines(c.Id)
}

func (c *SCluster) GetKubeconfig() (string, error) {
	// TODO: check kubeconfig
	if len(c.Kubeconfig) != 0 {
		return c.Kubeconfig, nil
	}
	kubeconfig, err := c.GetDriver().GetKubeconfig(c)
	if err != nil {
		return "", err
	}
	return kubeconfig, c.SetKubeconfig(kubeconfig)
}

func (c *SCluster) SetK8sVersion(version string) error {
	_, err := c.GetModelManager().TableSpec().Update(c, func() error {
		c.Version = version
		return nil
	})
	return err
}

func (c *SCluster) SetKubeconfig(kubeconfig string) error {
	_, err := c.GetModelManager().TableSpec().Update(c, func() error {
		c.Kubeconfig = kubeconfig
		return nil
	})
	return err
}

func (c *SCluster) GetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	conf, err := c.GetKubeconfig()
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(conf), "kubeconfig")
	return ret, nil
}

func (c *SCluster) GetAdminKubeconfig() (string, error) {
	return c.GetKubeconfig()
}

func (c *SCluster) GetK8sRestConfig() (*rest.Config, error) {
	kubeconfig, err := c.GetAdminKubeconfig()
	if err != nil {
		return nil, err
	}
	return k8s.GetK8sClientConfig([]byte(kubeconfig))
}

func (c *SCluster) GetK8sClient() (*kubernetes.Clientset, error) {
	config, err := c.GetK8sRestConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (c *SCluster) AllowPerformApplyAddons(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, data)
}

func (c *SCluster) PerformApplyAddons(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := c.StartApplyAddonsTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *SCluster) StartApplyAddonsTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterApplyAddonsTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) DeleteMachines(ctx context.Context, userCred mcclient.TokenCredential) error {
	machines, err := c.GetMachines()
	if err != nil {
		return err
	}
	var errgrp errgroup.Group
	for _, m := range machines {
		tmpM := m
		errgrp.Go(func() error {
			return tmpM.DoSyncDelete(ctx, userCred)
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	return nil
}

func (c *SCluster) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, data)
}

func (c *SCluster) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, c.StartSyncStatus(ctx, userCred, "")
}

func (c *SCluster) StartSyncStatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return c.GetDriver().StartSyncStatus(c, ctx, userCred, parentTaskId)
}

func (c *SCluster) AllowPerformAddMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, data)
}

func FetchMachinesByCreateData(cluster *SCluster, data []*types.CreateMachineData) ([]manager.IMachine, error) {
	ret := make([]manager.IMachine, 0)
	ms, err := cluster.GetMachines()
	if err != nil {
		return nil, err
	}
	for _, d := range data {
		for _, m := range ms {
			if d.ResourceId == m.GetResourceId() {
				ret = append(ret, m)
				break
			}
		}
	}
	if len(data) != len(ret) {
		return nil, fmt.Errorf("Need %d created machines, only find: %d", len(data), len(ret))
	}
	return ret, nil
}

func FetchMachineIdsByCreateData(cluster *SCluster, data []*types.CreateMachineData) ([]string, error) {
	ms, err := FetchMachinesByCreateData(cluster, data)
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0)
	for _, m := range ms {
		ret = append(ret, m.GetId())
	}
	return ret, nil
}

func (c *SCluster) PerformAddMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(c.Status, []string{types.ClusterStatusRunning, types.ClusterStatusInit}) {
		return nil, httperrors.NewNotAcceptableError("Cluster status is %s", c.Status)
	}
	ms := []types.CreateMachineData{}
	if err := data.Unmarshal(&ms, "machines"); err != nil {
		return nil, err
	}
	machines := make([]*types.CreateMachineData, len(ms))
	for i := range ms {
		machines[i] = &ms[i]
	}
	driver := c.GetDriver()
	if err := driver.ValidateAddMachines(ctx, userCred, c, machines); err != nil {
		return nil, err
	}

	if err := driver.CreateMachines(ctx, userCred, c, machines); err != nil {
		return nil, err
	}

	ids, err := FetchMachineIdsByCreateData(c, machines)
	if err != nil {
		return nil, err
	}

	return nil, c.StartDeployMachinesTask(ctx, userCred, ids, "")
}

const (
	MachinesDeployIdsKey = "MachineIds"
)

func (c *SCluster) StartDeployMachinesTask(ctx context.Context, userCred mcclient.TokenCredential, machineIds []string, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewStringArray(machineIds), MachinesDeployIdsKey)
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeployMachinesTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) AllowPerformDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, data)
}

func (c *SCluster) PerformDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	machinesData, err := data.(*jsonutils.JSONDict).GetArray("machines")
	if err != nil {
		return nil, httperrors.NewInputParameterError("NotFound machines data: %v", err)
	}
	machines := []manager.IMachine{}
	for _, obj := range machinesData {
		id, err := obj.GetString()
		if err != nil {
			return nil, err
		}
		machineObj, err := manager.MachineManager().FetchMachineByIdOrName(userCred, id)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Not found node by id: %s", id)
		}
		machines = append(machines, machineObj)
	}
	if len(machines) == 0 {
		return nil, httperrors.NewInputParameterError("Machines id is empty")
	}
	nowCnt, err := c.GetMachinesCount()
	if err != nil {
		return nil, err
	}
	// delete all machines
	if nowCnt == len(machines) {
		if err := c.CheckPVCEmpty(); err != nil {
			return nil, err
		}
	}
	driver := c.GetDriver()
	if err := driver.ValidateDeleteMachines(ctx, userCred, c, machines); err != nil {
		return nil, err
	}
	return nil, c.StartDeleteMachinesTask(ctx, userCred, machines, data.(*jsonutils.JSONDict), "")
}

func (c *SCluster) StartDeleteMachinesTask(ctx context.Context, userCred mcclient.TokenCredential, ms []manager.IMachine, data *jsonutils.JSONDict, parentTaskId string) error {
	for _, m := range ms {
		m.SetStatus(userCred, types.MachineStatusDeleting, "ClusterDeleteMachinesTask")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeleteMachinesTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
