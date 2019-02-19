package clusters

import (
	"context"
	"database/sql"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/k8s"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
	//"yunion.io/x/yunion-kube/pkg/models/clusters/drivers"
)

var ClusterManager *SClusterManager

func init() {
	ClusterManager = &SClusterManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SCluster{}, "kubeclusters_tbl", "kubecluster", "kubeclusters"),
	}
	manager.RegisterClusterManager(ClusterManager)
}

type SClusterManager struct {
	db.SVirtualResourceBaseManager
}

type SCluster struct {
	db.SVirtualResourceBase
	ClusterType   string            `nullable:"false" create:"required" list:"user"`
	CloudType     string            `nullable:"false" create:"required" list:"user"`
	Mode          string            `nullable:"false" create:"required" list:"user"`
	Provider      string            `nullable:"false" create:"required" list:"user"`
	ServiceCidr   string            `nullable:"false" create:"required" list:"user"`
	ServiceDomain string            `nullable:"false" create:"required" list:"user"`
	PodCidr       string            `nullable:"true" create:"optional" list:"user"`
	Version       string            `nullable:"true" create:"optional" list:"user"`
	Namespace     string            `nullable:"true" create:"optional" list:"user"`
	Ha            tristate.TriState `nullable:"true" create:"required" list:"user"`
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
	if !utils.IsInStringArray(providerType, []string{
		string(types.ProviderTypeOnecloud),
		string(types.ProviderTypeSystem),
	}) {
		return nil, httperrors.NewInputParameterError("Invalid provider type: %q", providerType)
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

func (m *SClusterManager) GetCluster(id string) (*SCluster, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (c *SCluster) GetDriver() IClusterDriver {
	return GetDriver(types.ProviderType(c.Provider))
}

func (c *SCluster) ValidateAddMachine(machine *types.Machine) error {
	if !utils.IsInStringArray(c.Status, []string{types.ClusterStatusInit, types.ClusterStatusCreating, types.ClusterStatusReady}) {
		return httperrors.NewNotAcceptableError("Can't add machine when cluster status is %s", c.Status)
	}
	driver := c.GetDriver()
	return driver.ValidateAddMachine(ClusterManager, machine)
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

func (c *SCluster) ValidateDeleteCondition(ctx context.Context) error {
	/*machines, err := manager.MachineManager().GetMachines(c.Id)
	if err != nil {
		return err
	}
	if len(machines) > 0 {
		return httperrors.NewNotEmptyError("Not an empty cluster")
	}*/
	return c.GetDriver().ValidateDeleteCondition()
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

func (c *SCluster) GetControlplaneMachine() (manager.IMachine, error) {
	machines, err := c.GetMachines()
	if err != nil {
		return nil, err
	}
	for _, m := range machines {
		if m.IsControlplane() && m.IsRunning() {
			return m, nil
		}
	}
	return nil, fmt.Errorf("Not found a ready controlplane machine")
}

func (c *SCluster) GetMachines() ([]manager.IMachine, error) {
	return manager.MachineManager().GetMachines(c.Id)
}

func (c *SCluster) GetKubeConfig() (string, error) {
	return c.GetDriver().GetKubeconfig(c)
}

func (c *SCluster) GetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	conf, err := c.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(conf), "kubeconfig")
	return ret, nil
}

func (c *SCluster) GetAdminKubeconfig() (string, error) {
	return c.GetKubeConfig()
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
	for _, m := range machines {
		// TODO: parralle
		if err := m.DoSyncDelete(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}
