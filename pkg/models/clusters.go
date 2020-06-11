package models

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	ocapis "yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/api/constants"
	"yunion.io/x/yunion-kube/pkg/clientv2"
	k8sutil "yunion.io/x/yunion-kube/pkg/k8s/util"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/utils/certificates"
	"yunion.io/x/yunion-kube/pkg/utils/tokens"
)

var ClusterManager *SClusterManager

func init() {
	ClusterManager = &SClusterManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SCluster{},
			"kubeclusters_tbl",
			"kubecluster",
			"kubeclusters",
		),
	}
	manager.RegisterClusterManager(ClusterManager)
	ClusterManager.SetVirtualObject(ClusterManager)
}

// +onecloud:swagger-gen-model-singular=kubecluster
type SClusterManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SCluster struct {
	db.SSharableVirtualResourceBase

	ClusterType     string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	CloudType       string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ResourceType    string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Mode            string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Provider        string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ServiceCidr     string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ServiceDomain   string               `width:"128" charset:"ascii" nullable:"false" create:"required" list:"user"`
	PodCidr         string               `width:"36" charset:"ascii" nullable:"true" create:"optional" list:"user"`
	Version         string               `width:"128" charset:"ascii" nullable:"false" create:"optional" list:"user"`
	Ha              tristate.TriState    `nullable:"true" create:"required" list:"user"`
	ImageRepository jsonutils.JSONObject `nullable:"true" create:"optional" list:"user"`

	// kubernetes config
	Kubeconfig string `nullable:"true" charset:"utf8" create:"optional"`

	// kubernetes api server endpoint
	ApiServer string `width:"256" nullable:"true" charset:"ascii" create:"optional" list:"user"`
}

func (m *SClusterManager) InitializeData() error {
	clusters := []SCluster{}
	q := m.Query().IsNullOrEmpty("resource_type")
	err := db.FetchModelObjects(m, q, &clusters)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		tmp := &cluster
		db.Update(tmp, func() error {
			tmp.ResourceType = string(api.ClusterResourceTypeHost)
			return nil
		})
	}
	return nil
}

func (m *SClusterManager) GetSystemCluster() (*SCluster, error) {
	clusters := m.Query().SubQuery()
	q := clusters.Query().Filter(sqlchemy.Equals(clusters.Field("provider"), string(api.ProviderTypeSystem)))
	q = q.Filter(sqlchemy.IsTrue(q.Field("is_system")))
	objs := make([]SCluster, 0)
	err := db.FetchModelObjects(m, q, &objs)
	if err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		// return nil, httperrors.NewNotFoundError("Not found default system cluster")
		return nil, nil
	}
	if len(objs) != 1 {
		return nil, httperrors.NewDuplicateResourceError("Found %d system cluster", len(objs))
	}
	sysCluster := objs[0]
	return &sysCluster, nil
}

func (m *SClusterManager) GetSystemClusterConfig() (*rest.Config, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

const (
	SystemClusterName = "system-default"
	NamespaceOneCloud = "onecloud"
)

type k8sInfo struct {
	ApiServer  string
	Kubeconfig string
}

func (m *SClusterManager) GetSystemClusterK8SInfo() (*k8sInfo, error) {
	restCfg, err := m.GetSystemClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get rest config")
	}
	kubeconfig, err := m.GetSystemClusterKubeconfig(restCfg.Host, restCfg)
	if err != nil {
		return nil, errors.Wrap(err, "generate k8s kubeconfig")
	}
	return &k8sInfo{
		ApiServer:  restCfg.Host,
		Kubeconfig: kubeconfig,
	}, nil
}

func (m *SClusterManager) GetSystemClusterCreateData() (*api.ClusterCreateInput, error) {
	createData := &api.ClusterCreateInput{
		Name:        SystemClusterName,
		ClusterType: string(api.ClusterTypeDefault),
		CloudType:   string(api.CloudTypePrivate),
		Mode:        string(api.ModeTypeImport),
		Provider:    string(api.ProviderTypeSystem),
	}
	k8sInfo, err := m.GetSystemClusterK8SInfo()
	if err != nil {
		return nil, errors.Wrap(err, "get k8s info")
	}
	createData.ApiServer = k8sInfo.ApiServer
	createData.Kubeconfig = k8sInfo.Kubeconfig
	return createData, nil
}

func (m *SClusterManager) RegisterSystemCluster() error {
	sysCluster, err := m.GetSystemCluster()
	if err != nil {
		return errors.Wrap(err, "get system cluster")
	}
	userCred := GetAdminCred()
	if sysCluster == nil {
		// create system cluster
		createData, err := m.GetSystemClusterCreateData()
		if err != nil {
			return errors.Wrap(err, "get cluster create data")
		}
		obj, err := db.DoCreate(m, context.TODO(), userCred, nil, createData.JSON(createData), userCred)
		if err != nil {
			return errors.Wrap(err, "create cluster")
		}
		sysCluster = obj.(*SCluster)
	}
	// update system cluster
	k8sInfo, err := m.GetSystemClusterK8SInfo()
	if err != nil {
		return errors.Wrap(err, "get k8s info")
	}
	if _, err := db.Update(sysCluster, func() error {
		if sysCluster.ApiServer != k8sInfo.ApiServer {
			sysCluster.ApiServer = k8sInfo.ApiServer
		}
		if sysCluster.Kubeconfig != k8sInfo.Kubeconfig {
			sysCluster.Kubeconfig = k8sInfo.Kubeconfig
		}
		if !sysCluster.IsSystem {
			sysCluster.IsSystem = true
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "update system cluster")
	}
	return sysCluster.StartSyncStatus(context.TODO(), userCred, "")
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
	return GetAdminSession()
}

func (m *SClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterListInput) (*sqlchemy.SQuery, error) {
	return m.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
}

func (m *SClusterManager) CreateCluster(ctx context.Context, userCred mcclient.TokenCredential, data api.ClusterCreateInput) (manager.ICluster, error) {
	input := jsonutils.Marshal(data)
	obj, err := db.DoCreate(m, ctx, userCred, nil, input, userCred)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (m *SClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var (
		clusterType  string
		cloudType    string
		modeType     string
		providerType string
	)

	clusterType = SetJSONDataDefault(data, "cluster_type", string(api.ClusterTypeDefault))
	if !utils.IsInStringArray(clusterType, []string{string(api.ClusterTypeDefault)}) {
		return nil, httperrors.NewInputParameterError("Invalid cluster type: %q", clusterType)
	}

	cloudType = SetJSONDataDefault(data, "cloud_type", string(api.CloudTypePrivate))
	if !utils.IsInStringArray(cloudType, []string{string(api.CloudTypePrivate)}) {
		return nil, httperrors.NewInputParameterError("Invalid cloud type: %q", cloudType)
	}

	resType := SetJSONDataDefault(data, "resource_type", string(api.ClusterResourceTypeHost))
	if err := m.ValidateResourceType(resType); err != nil {
		return nil, err
	}

	modeType = SetJSONDataDefault(data, "mode", string(api.ModeTypeSelfBuild))
	if !utils.IsInStringArray(modeType, []string{
		string(api.ModeTypeSelfBuild),
		string(api.ModeTypeImport),
	}) {
		return nil, httperrors.NewInputParameterError("Invalid mode type: %q", modeType)
	}

	providerType = SetJSONDataDefault(data, "provider", string(api.ProviderTypeOnecloud))
	if err := m.ValidateProviderType(providerType); err != nil {
		return nil, err
	}

	driver, err := GetDriverWithError(
		api.ModeType(modeType),
		api.ProviderType(providerType),
		api.ClusterResourceType(resType),
	)
	if err != nil {
		return nil, err
	}

	serviceCidr := SetJSONDataDefault(data, "service_cidr", api.DefaultServiceCIDR)
	if _, err := netutils.NewIPV4Prefix(serviceCidr); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid service CIDR: %q", serviceCidr)
	}

	serviceDomain := SetJSONDataDefault(data, "service_domain", api.DefaultServiceDomain)
	if len(serviceDomain) == 0 {
		return nil, httperrors.NewInputParameterError("service domain must provided")
	}

	podCidr := SetJSONDataDefault(data, "pod_cidr", api.DefaultPodCIDR)
	if _, err := netutils.NewIPV4Prefix(serviceCidr); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid pod CIDR: %q", podCidr)
	}

	if jsonutils.QueryBoolean(data, "ha", false) {
		data.Set("ha", jsonutils.JSONTrue)
	} else {
		data.Set("ha", jsonutils.JSONFalse)
	}

	res := api.ClusterCreateInput{}
	if err := data.Unmarshal(&res); err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal: %v", err)
	}

	if res.Provider != string(api.ProviderTypeSystem) && driver.NeedCreateMachines() && len(res.Machines) == 0 {
		return nil, httperrors.NewInputParameterError("Machines desc not provider")
	}

	var machineResType api.MachineResourceType
	for _, m := range res.Machines {
		if len(m.ResourceType) == 0 {
			return nil, httperrors.NewInputParameterError("Machine resource type is empty")
		}
		if len(machineResType) == 0 {
			machineResType = api.MachineResourceType(m.ResourceType)
		}
		if string(machineResType) != m.ResourceType {
			return nil, httperrors.NewInputParameterError("Machine resource type must same")
		}
	}

	if err := driver.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}

	versions := driver.GetK8sVersions()
	if len(versions) > 0 {
		defaultVersion := versions[0]
		version := SetJSONDataDefault(data, "version", defaultVersion)
		if !utils.IsInStringArray(version, versions) {
			return nil, httperrors.NewInputParameterError("Invalid version: %q, choose one from %v", version, versions)
		}
	}

	imageRepo := res.ImageRepository
	if imageRepo != nil {
		if imageRepo.Url == "" {
			return nil, httperrors.NewNotEmptyError("image_repository.url is empty, use format: 'registry.hub.docker.com/yunion'")
		}
		if _, err := m.GetRegistryUrlByRepoUrl(imageRepo.Url); err != nil {
			return nil, err
		}
	}

	input := ocapis.VirtualResourceCreateInput{}
	if err := data.Unmarshal(&input); err != nil {
		return nil, err
	}
	input, err = m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (m *SClusterManager) AllowGetPropertyK8sVersions(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (m *SClusterManager) ValidateProviderType(providerType string) error {
	if !utils.IsInStringArray(providerType, []string{
		string(api.ProviderTypeOnecloud),
		string(api.ProviderTypeSystem),
		string(api.ProviderTypeExternal),
	}) {
		return httperrors.NewInputParameterError("Invalid provider type: %q", providerType)
	}
	return nil
}

func (m *SClusterManager) ValidateResourceType(resType string) error {
	if !utils.IsInStringArray(resType, []string{
		string(api.ClusterResourceTypeHost),
		string(api.ClusterResourceTypeGuest),
		string(api.ClusterResourceTypeUnknown),
	}) {
		return httperrors.NewInputParameterError("Invalid cluster resource type: %q", resType)
	}
	return nil
}

func (m *SClusterManager) GetDriverByQuery(query jsonutils.JSONObject) (IClusterDriver, error) {
	modeType, _ := query.GetString("mode")
	providerType, _ := query.GetString("provider")
	resType, _ := query.GetString("resource_type")
	if err := m.ValidateProviderType(providerType); err != nil {
		return nil, err
	}
	if len(resType) == 0 {
		resType = string(api.ClusterResourceTypeHost)
	}
	if err := m.ValidateResourceType(resType); err != nil {
		return nil, err
	}
	driver := GetClusterDriver(
		api.ModeType(modeType),
		api.ProviderType(providerType),
		api.ClusterResourceType(resType))
	return driver, nil
}

func (m *SClusterManager) GetPropertyK8sVersions(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	SetJSONDataDefault(query.(*jsonutils.JSONDict), "mode", string(api.ModeTypeSelfBuild))
	driver, err := m.GetDriverByQuery(query)
	if err != nil {
		return nil, err
	}
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
	clusters := m.Query().SubQuery()
	q := clusters.Query()
	q = q.Filter(sqlchemy.Equals(clusters.Field("status"), api.ClusterStatusRunning))
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	if cnt <= 0 {
		return false, nil
	}
	return true, nil
}

func (m *SClusterManager) AllowGetPropertyUsableInstances(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsAllow(rbacutils.ScopeSystem, m.KeywordPlural(), policy.PolicyActionGet, "usable-instances")
}

func (m *SClusterManager) GetPropertyUsableInstances(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	SetJSONDataDefault(query.(*jsonutils.JSONDict), "mode", string(api.ModeTypeSelfBuild))
	driver, err := m.GetDriverByQuery(query)
	if err != nil {
		return nil, err
	}
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

/*func (m *SClusterManager) GetNonSystemClusters() ([]manager.ICluster, error) {
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
}*/

func (m *SClusterManager) GetRunningClusters() ([]manager.ICluster, error) {
	return m.GetClustersByStatus(api.ClusterStatusRunning)
}

func (m *SClusterManager) GetClustersByStatus(status ...string) ([]manager.ICluster, error) {
	clusters := m.Query().SubQuery()
	q := clusters.Query()
	q = q.Filter(sqlchemy.In(clusters.Field("status"), status))
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

func (m *SClusterManager) GetCluster(id string) (*SCluster, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (m *SClusterManager) ClusterHealthCheckTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	clusters, err := m.GetClustersByStatus(
		api.ClusterStatusRunning,
		api.ClusterStatusLost,
		//types.ClusterStatusUnknown,
	)
	if err != nil {
		log.Errorf("ClusterHealthCheckTask get clusters: %v", err)
		return
	}
	for _, obj := range clusters {
		c := obj.(*SCluster)
		if err := c.IsHealthy(); err == nil {
			if c.Status != api.ClusterStatusRunning {
				c.SetStatus(userCred, api.ClusterStatusRunning, "by health check cronjob")
			}
			continue
		} else {
			c.SetStatus(userCred, api.ClusterStatusLost, err.Error())
		}
	}
}

func (c *SCluster) GetDriver() IClusterDriver {
	return GetClusterDriver(
		api.ModeType(c.Mode),
		api.ProviderType(c.Provider),
		api.ClusterResourceType(c.ResourceType))
}

func (c *SCluster) GetMachinesCount() (int, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return 0, err
	}
	return len(ms), nil
}

func (man *SClusterManager) GetImageRepository(input *api.ImageRepository) *api.ImageRepository {
	ret := &api.ImageRepository{
		Url: constants.DefaultRegistryMirror,
	}
	if input == nil {
		return ret
	}
	if input.Url != "" {
		ret.Url = input.Url
	}
	ret.Insecure = input.Insecure
	return ret
}

func (c *SCluster) GetImageRepository() (*api.ImageRepository, error) {
	ret := new(api.ImageRepository)
	if c.ImageRepository == nil {
		return ClusterManager.GetImageRepository(nil), nil
	}
	if err := c.ImageRepository.Unmarshal(ret); err != nil {
		return nil, err
	}
	return ClusterManager.GetImageRepository(ret), nil
}

func (c *SCluster) IsHealthy() error {
	cli, err := c.GetK8sClient()
	if err != nil {
		return err
	}
	if _, err := cli.Discovery().ServerVersion(); err != nil {
		return err
	}
	return nil
}

func (m *SClusterManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []*jsonutils.JSONDict {
	rows := make([]*jsonutils.JSONDict, len(objs))
	virtRows := m.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		rows[i] = jsonutils.Marshal(virtRows[i]).(*jsonutils.JSONDict)
		rows[i] = objs[i].(*SCluster).moreExtraInfo(rows[i])
	}
	return rows
}

func (c *SCluster) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (*jsonutils.JSONDict, error) {
	extra, err := c.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query, isList)
	if err != nil {
		return nil, err
	}

	return c.moreExtraInfo(jsonutils.Marshal(extra).(*jsonutils.JSONDict)), nil
}

func (c *SCluster) moreExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	if cnt, err := c.GetMachinesCount(); err != nil {
		log.Errorf("GetMachines error: %v", err)
	} else {
		extra.Add(jsonutils.NewInt(int64(cnt)), "machines")
	}
	return extra
}

type CertificatesGroup struct {
	CAKeyPair           *SX509KeyPair
	EtcdCAKeyPair       *SX509KeyPair
	FrontProxyCAKeyPair *SX509KeyPair
	SAKeyPair           *SX509KeyPair
}

func (c *SCluster) GetCertificatesGroup() (*CertificatesGroup, error) {
	caKp, err := c.GetCAKeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "get CAKeyPair")
	}
	etcdKp, err := c.GetEtcdCAKeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "get EtcdCAKeyPair")
	}
	fpKp, err := c.GetFrontProxyCAKeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "get FrontProxyCAKeyPair")
	}
	saKp, err := c.GetSAKeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "get ServiceAccount KeyPair")
	}
	return &CertificatesGroup{
		CAKeyPair:           caKp,
		EtcdCAKeyPair:       etcdKp,
		FrontProxyCAKeyPair: fpKp,
		SAKeyPair:           saKp,
	}, nil
}

func (man *SClusterManager) GetRegistryUrlByRepoUrl(imageRepo string) (string, error) {
	rets := strings.Split(imageRepo, "/")
	if len(rets) != 2 {
		return "", httperrors.NewInputParameterError("Invalid image repository format %s, use like: 'docker.io/yunion'", imageRepo)
	}
	return rets[0], nil
}

func (c *SCluster) GetDefaultMachineDockerConfig(imageRepo *api.ImageRepository) (*api.DockerConfig, error) {
	ret := new(api.DockerConfig)
	if imageRepo.Insecure {
		reg, err := ClusterManager.GetRegistryUrlByRepoUrl(imageRepo.Url)
		if err != nil {
			return nil, err
		}
		ret.InsecureRegistries = []string{reg}
	}
	return ret, nil
}

func (c *SCluster) FillMachinePrepareInput(input *api.MachinePrepareInput) (*api.MachinePrepareInput, error) {
	cg, err := c.GetCertificatesGroup()
	if err != nil {
		return nil, errors.Wrap(err, "get certificates group")
	}
	input.CAKeyPair = cg.CAKeyPair.ToKeyPair()
	input.EtcdCAKeyPair = cg.EtcdCAKeyPair.ToKeyPair()
	input.FrontProxyCAKeyPair = cg.FrontProxyCAKeyPair.ToKeyPair()
	input.SAKeyPair = cg.SAKeyPair.ToKeyPair()
	if !input.FirstNode {
		bootstrapToken, err := c.GetNodeJoinToken()
		if err != nil {
			return nil, errors.Wrapf(err, "get %s node join token", input.Role)
		}
		input.BootstrapToken = bootstrapToken
	}
	imageRepo, err := c.GetImageRepository()
	if err != nil {
		return nil, err
	}
	input.Config.ImageRepository = imageRepo
	dockerConfig, err := c.GetDefaultMachineDockerConfig(imageRepo)
	if err != nil {
		return nil, err
	}
	input.Config.DockerConfig = dockerConfig
	// TODO: support lb
	return input, nil
}

func (c *SCluster) GetNodeJoinToken() (string, error) {
	kubeConfig, err := c.GetKubeconfig()
	if err != nil {
		return "", errors.Wrapf(err, "failed to retrieve kubeconfig for cluster %q", c.GetName())
	}
	controlPlaneURL, err := c.GetControlPlaneUrl()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get controlPlaneURL")
	}
	clientConfig, err := clientcmd.BuildConfigFromKubeconfigGetter(controlPlaneURL, func() (*clientcmdapi.Config, error) {
		return clientcmd.Load([]byte(kubeConfig))
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get client config for cluster at %q", controlPlaneURL)
	}

	coreClient, err := corev1.NewForConfig(clientConfig)
	if err != nil {
		return "", errors.Wrapf(err, "failed to initialize new corev1 client")
	}

	bootstrapToken, err := tokens.NewBootstrap(coreClient, 24*time.Hour)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create new bootstrap token")
	}
	return bootstrapToken, nil
}

func (c *SCluster) AttachKeypair(ctx context.Context, userCred mcclient.TokenCredential, keypair *SX509KeyPair) error {
	attached, err := c.IsAttachKeypair(keypair)
	if err != nil {
		return errors.Wrapf(err, "check keypair %s attached to cluster %s", keypair.GetName(), c.GetName())
	}
	if attached {
		return errors.Errorf("Cluster %s already attached keypair %s", c.GetName(), keypair.GetName())
	}
	model, err := db.NewModelObject(ClusterX509KeyPairManager)
	if err != nil {
		return errors.Wrapf(err, "new cluster %s keypair %s obj", c.GetName(), keypair.GetName())
	}

	clusterKeypair := model.(*SClusterX509KeyPair)
	clusterKeypair.ClusterId = c.GetId()
	clusterKeypair.KeypairId = keypair.GetId()
	clusterKeypair.User = keypair.User
	return ClusterX509KeyPairManager.TableSpec().Insert(clusterKeypair)
}

func (c *SCluster) IsAttachKeypair(kp *SX509KeyPair) (bool, error) {
	q := ClusterX509KeyPairManager.Query().Equals("keypair_id", kp.GetId()).Equals("cluster_id", c.GetId())
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (c *SCluster) GenerateCertificates(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !c.GetDriver().NeedGenerateCertificate() {
		return nil
	}
	clusterCAKeyPair, err := X509KeyPairManager.GenerateCertificates(ctx, userCred, c, api.ClusterCA)
	if err != nil {
		return errors.Wrapf(err, "Generate %s certificate", api.ClusterCA)
	}
	infof := func(kp *SX509KeyPair) {
		log.Infof("Generate cluster %s %s certificate", c.GetName(), kp.GetName())
	}
	infof(clusterCAKeyPair)
	etcdCAKeyPair, err := X509KeyPairManager.GenerateCertificates(ctx, userCred, c, api.EtcdCA)
	if err != nil {
		return errors.Wrapf(err, "Generate %s certificate", api.EtcdCA)
	}
	infof(etcdCAKeyPair)
	fpCAKeyPair, err := X509KeyPairManager.GenerateCertificates(ctx, userCred, c, api.FrontProxyCA)
	if err != nil {
		return errors.Wrapf(err, "Generate %s certificate", api.FrontProxyCA)
	}
	infof(fpCAKeyPair)
	saKeyPair, err := X509KeyPairManager.GenerateServiceAccountKeys(ctx, userCred, c, api.ServiceAccountCA)
	if err != nil {
		return errors.Wrapf(err, "Generate ServiceAccount key %s", api.ServiceAccountCA)
	}
	infof(saKeyPair)
	return nil
}

func (c *SCluster) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if err := c.StartClusterCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartClusterCreateTask error: %v", err)
	}
}

func (c *SCluster) StartClusterCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	c.SetStatus(userCred, api.ClusterStatusCreating, "")
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
	//if err := c.CheckPVCEmpty(); err != nil {
	//return err
	//}
	return nil
}

func (c *SCluster) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Cluster delete do nothing")
	return nil
}

func (c *SCluster) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := X509KeyPairManager.DeleteKeyPairsByCluster(ctx, userCred, c); err != nil {
		return errors.Wrapf(err, "DeleteKeyPairsByCluster")
	}
	if err := c.DeleteAllComponents(ctx, userCred); err != nil {
		return errors.Wrapf(err, "DeleteClusterComponent")
	}
	return c.SVirtualResourceBase.Delete(ctx, userCred)
}

func (c *SCluster) DeleteAllComponents(ctx context.Context, userCred mcclient.TokenCredential) error {
	cs, err := c.GetClusterComponents()
	if err != nil {
		return err
	}
	for _, cp := range cs {
		comp, err := cp.GetComponent()
		if err != nil {
			return err
		}
		if err := cp.Detach(ctx, userCred); err != nil {
			return err
		}
		if err := comp.Delete(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (c *SCluster) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return c.StartClusterDeleteTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (c *SCluster) StartClusterDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := c.SetStatus(userCred, api.ClusterStatusDeleting, ""); err != nil {
		return err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeleteTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) allowPerformAction(userCred mcclient.TokenCredential, action string) bool {
	return db.IsProjectAllowPerform(userCred, c, action)
}

func (c *SCluster) allowGetSpec(userCred mcclient.TokenCredential, spec string) bool {
	return db.IsProjectAllowGetSpec(userCred, c, spec)
}

func (c *SCluster) AllowPerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "terminate")
}

func (c *SCluster) PerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := c.ValidateDeleteCondition(ctx); err != nil {
		return nil, err
	}
	return nil, c.RealDelete(ctx, userCred)
}

func (c *SCluster) AllowGetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.allowGetSpec(userCred, "kubeconfig")
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

func (c *SCluster) GetMachinesByRole(role string) ([]manager.IMachine, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return nil, err
	}
	ret := make([]manager.IMachine, 0)
	for _, m := range ms {
		if m.GetRole() == role {
			ret = append(ret, m)
		}
	}
	return ret, nil
}

func (c *SCluster) getKeyPairByUser(user string) (*SX509KeyPair, error) {
	return ClusterX509KeyPairManager.GetKeyPairByClusterUser(c.GetId(), user)
}

func (c *SCluster) GetCAKeyPair() (*SX509KeyPair, error) {
	return c.getKeyPairByUser(api.ClusterCA)
}

func (c *SCluster) GetEtcdCAKeyPair() (*SX509KeyPair, error) {
	return c.getKeyPairByUser(api.EtcdCA)
}

func (c *SCluster) GetFrontProxyCAKeyPair() (*SX509KeyPair, error) {
	return c.getKeyPairByUser(api.FrontProxyCA)
}

func (c *SCluster) GetSAKeyPair() (*SX509KeyPair, error) {
	return c.getKeyPairByUser(api.ServiceAccountCA)
}

func (c *SCluster) GetKubeconfig() (string, error) {
	if len(c.Kubeconfig) != 0 {
		return c.Kubeconfig, nil
	}
	//kubeconfig, err := c.GetDriver().GetKubeconfig(c)
	kubeconfig, err := c.GetKubeconfigByCerts()
	if err != nil {
		return "", err
	}
	return kubeconfig, c.SetKubeconfig(kubeconfig)
}

func (c *SCluster) GetClientV2() (*clientv2.Client, error) {
	kubeconfig, err := c.GetKubeconfig()
	if err != nil {
		return nil, err
	}
	return clientv2.NewClient(kubeconfig)
}

func (c *SCluster) GetKubeconfigByCerts() (string, error) {
	caKpObj, err := c.GetCAKeyPair()
	if err != nil {
		return "", errors.Wrap(err, "Get CA key pair")
	}
	caKp := caKpObj.ToKeyPair()
	cert, err := certificates.DecodeCertPEM(caKp.Cert)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode CA Cert")
	} else if cert == nil {
		return "", errors.Errorf("certificate not found")
	}

	key, err := certificates.DecodePrivateKeyPEM(caKp.Key)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode private key")
	} else if key == nil {
		return "", errors.Errorf("key not foudn in status")
	}
	controlPlaneURL, err := c.GetControlPlaneUrl()
	if err != nil {
		return "", errors.Wrap(err, "failed to get controlPlaneURL")
	}

	cfg, err := certificates.NewKubeconfig(c.GetName(), controlPlaneURL, cert, key)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate a kubeconfig")
	}

	yaml, err := clientcmd.Write(*cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize config to yaml")
	}

	return string(yaml), nil
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

func setK8sConfigField(c *rest.Config, tr func(rt http.RoundTripper) http.RoundTripper) *rest.Config {
	if tr != nil {
		c.WrapTransport = tr
	}
	c.Timeout = time.Second * 30
	return c
}

func (c *SCluster) GetK8sClientConfig(kubeConfig []byte) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if kubeConfig != nil {
		apiconfig, err := clientcmd.Load(kubeConfig)
		if err != nil {
			return nil, err
		}

		clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.Errorf("kubeconfig value is nil")
	}
	if err != nil {
		return nil, errors.Errorf("create kubernetes config failed: %v", err)
	}
	return config, nil
}

func (c *SCluster) GetK8sRestConfig() (*rest.Config, error) {
	kubeconfig, err := c.GetAdminKubeconfig()
	if err != nil {
		return nil, err
	}
	config, err := c.GetK8sClientConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}
	return setK8sConfigField(config, func(rt http.RoundTripper) http.RoundTripper {
		switch rt.(type) {
		case *http.Transport:
			rt.(*http.Transport).DisableKeepAlives = true
		}
		return rt
	}), nil
}

func (c *SCluster) GetK8sClient() (*kubernetes.Clientset, error) {
	config, err := c.GetK8sRestConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (c *SCluster) AllowPerformApplyAddons(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "apply-addons")
}

func (c *SCluster) PerformApplyAddons(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := c.StartApplyAddonsTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *SCluster) AllowGetDetailsAddons(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.AllowGetDetails(ctx, userCred, query)
}

func (c *SCluster) GetDetailsAddons(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	addons, err := c.GetDriver().GetAddonsManifest(c)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(addons), "addons")
	return ret, nil
}

func (c *SCluster) StartApplyAddonsTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterApplyAddonsTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "syncstatus")
}

func (c *SCluster) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, c.StartSyncStatus(ctx, userCred, "")
}

func (c *SCluster) StartSyncStatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return c.GetDriver().StartSyncStatus(c, ctx, userCred, parentTaskId)
}

func (c *SCluster) AllowPerformAddMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "add-machines")
}

func (c *SCluster) ValidateAddMachines(ctx context.Context, userCred mcclient.TokenCredential, ms []api.CreateMachineData) ([]*api.CreateMachineData, error) {
	machines := make([]*api.CreateMachineData, len(ms))
	for i := range ms {
		machines[i] = &ms[i]
	}
	driver := c.GetDriver()
	imageRepo, err := c.GetImageRepository()
	if err != nil {
		return nil, err
	}
	if err := driver.ValidateCreateMachines(ctx, userCred, c, imageRepo, machines); err != nil {
		return nil, err
	}
	return machines, nil
}

func (c *SCluster) PerformAddMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ms := []api.CreateMachineData{}
	if err := data.Unmarshal(&ms, "machines"); err != nil {
		return nil, err
	}
	if !utils.IsInStringArray(c.Status, []string{api.ClusterStatusRunning, api.ClusterStatusInit}) {
		return nil, httperrors.NewNotAcceptableError("Cluster status is %s", c.Status)
	}

	machines, err := c.ValidateAddMachines(ctx, userCred, ms)
	if err != nil {
		return nil, err
	}

	return nil, c.StartCreateMachinesTask(ctx, userCred, machines, "")
}

func (c *SCluster) NeedControlplane() (bool, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return false, errors.Wrapf(err, "get cluster %s machines", c.GetName())
	}
	if len(ms) == 0 {
		return true, nil
	}
	return false, nil
}

func (c *SCluster) StartCreateMachinesTask(ctx context.Context, userCred mcclient.TokenCredential, machines []*api.CreateMachineData, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.Marshal(machines), "machines")
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterCreateMachinesTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, ms []*api.CreateMachineData, task taskman.ITask) error {
	drv := c.GetDriver()
	machines, err := drv.CreateMachines(ctx, userCred, c, ms)
	if err != nil {
		return err
	}
	return drv.RequestDeployMachines(ctx, userCred, c, machines, task)
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
	return c.allowPerformAction(userCred, "delete-machines")
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
	if data == nil {
		data = jsonutils.NewDict()
	}
	mids := []jsonutils.JSONObject{}
	for _, m := range ms {
		m.SetStatus(userCred, api.MachineStatusDeleting, "ClusterDeleteMachinesTask")
		mids = append(mids, jsonutils.NewString(m.GetId()))
	}
	data.Set("machines", jsonutils.NewArray(mids...))
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeleteMachinesTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) GetControlPlaneUrl() (string, error) {
	apiServerEndpoint, err := c.GetAPIServerEndpoint()
	if err != nil {
		return "", errors.Wrapf(err, "GetAPIServerEndpoint")
	}
	return fmt.Sprintf("https://%s:6443", apiServerEndpoint), nil
}

func (c *SCluster) GetAPIServer() (string, error) {
	if len(c.ApiServer) != 0 {
		return c.ApiServer, nil
	}

	apiServer, err := c.GetControlPlaneUrl()
	if err != nil {
		return "", err
	}
	return apiServer, c.SetAPIServer(apiServer)
}

func (c *SCluster) SetAPIServer(apiServer string) error {
	_, err := c.GetModelManager().TableSpec().Update(c, func() error {
		c.ApiServer = apiServer
		return nil
	})
	return err
}

func (c *SCluster) GetAPIServerEndpoint() (string, error) {
	m, err := c.getControlplaneMachine(false)
	if err != nil {
		return "", errors.Wrap(err, "get controlplane machine")
	}
	ip, err := m.GetPrivateIP()
	if err != nil {
		return "", errors.Wrapf(err, "get controlplane machine %s private ip", m.GetName())
	}
	return ip, nil
}

func (c *SCluster) GetPodCidr() string {
	return c.PodCidr
}

func (c *SCluster) GetServiceCidr() string {
	return c.ServiceCidr
}

func (c *SCluster) GetServiceDomain() string {
	return c.ServiceDomain
}

func (c *SCluster) GetVersion() string {
	return c.Version
}

func (c *SCluster) GetClusterComponents() ([]SClusterComponent, error) {
	cs := make([]SClusterComponent, 0)
	q := ClusterComponentManager.Query().Equals("cluster_id", c.GetId())
	if err := db.FetchModelObjects(ClusterComponentManager, q, &cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func (c *SCluster) GetComponents() ([]*SComponent, error) {
	cs, err := c.GetClusterComponents()
	if err != nil {
		return nil, err
	}
	ret := make([]*SComponent, 0)
	for _, cc := range cs {
		obj, err := cc.GetComponent()
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				continue
			}
			return nil, err
		}
		ret = append(ret, obj)
	}
	return ret, nil
}

func (c *SCluster) GetComponentByTypeNoError(cType string) (*SComponent, error) {
	cs, err := c.GetComponents()
	if err != nil {
		return nil, err
	}
	for _, comp := range cs {
		if comp.Type == cType {
			return comp, nil
		}
	}
	return nil, nil
}

func (c *SCluster) GetComponentByType(cType string) (*SComponent, error) {
	comp, err := c.GetComponentByTypeNoError(cType)
	if err != nil {
		return nil, err
	}
	if comp == nil {
		return nil, httperrors.NewNotFoundError("not found component by type %q", cType)
	}
	return comp, nil
}

func (c *SCluster) EnableComponent(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.ComponentCreateInput) error {
	comp, err := c.GetComponentByTypeNoError(input.Type)
	if err != nil {
		return err
	}
	if comp != nil {
		return comp.DoEnable(ctx, userCred, nil, "")
	}

	defer lockman.ReleaseObject(ctx, c)
	lockman.LockObject(ctx, c)

	comp, err = ComponentManager.CreateByCluster(ctx, userCred, c, input)
	if err != nil {
		return err
	}
	return nil
}

func (c *SCluster) AllowGetDetailsComponentsStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsProjectAllowGetSpec(userCred, c, "components-status")
}

func (c *SCluster) GetDetailsComponentsStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*api.ComponentsStatus, error) {
	return c.GetComponentsStatus()
}

func (c *SCluster) GetComponentsStatus() (*api.ComponentsStatus, error) {
	status := new(api.ComponentsStatus)
	drvs := ComponentManager.GetDrivers()
	for _, drv := range drvs {
		comp, err := c.GetComponentByTypeNoError(drv.GetType())
		if err != nil {
			return nil, errors.Wrapf(err, "cluster get component by type: %s", drv.GetType())
		}
		if comp == nil {
			// not created
			if err := drv.FetchStatus(c, comp, status); err != nil {
				return nil, err
			}
		} else {
			if err := drv.FetchStatus(c, comp, status); err != nil {
				return nil, err
			}
		}
	}
	return status, nil
}

func (c *SCluster) AllowGetDetailsComponentSetting(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsProjectAllowGetSpec(userCred, c, "component-setting")
}

func (c *SCluster) GetDetailsComponentSetting(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !query.Contains("type") {
		return nil, httperrors.NewInputParameterError("type not provided")
	}
	cType, _ := query.GetString("type")
	cs, err := c.GetComponentByType(cType)
	if err != nil {
		return nil, err
	}
	return cs.Settings, nil
}

func (c *SCluster) AllowPerformEnableComponent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "enable-component")
}

func (c *SCluster) PerformEnableComponent(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ComponentCreateInput) (jsonutils.JSONObject, error) {
	if err := c.EnableComponent(ctx, userCred, input); err != nil {
		log.Errorf("enable comp error: %v", err)
		return nil, err
	}
	comp, err := c.GetComponentByType(input.Type)
	if err != nil {
		return nil, err
	}
	ret, err := comp.GetExtraDetails(ctx, userCred, query, false)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(ret), nil
}

func (c *SCluster) AllowPerformDisableComponent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "disable-component")
}

func (c *SCluster) PerformDisableComponent(ctx context.Context, userCred mcclient.TokenCredential, query, input api.ComponentDeleteInput) (jsonutils.JSONObject, error) {
	comp, err := c.GetComponentByType(input.Type)
	if err != nil {
		return nil, err
	}
	return nil, comp.DoDisable(ctx, userCred, input.JSON(input), "")
}

func (c *SCluster) AllowPerformDeleteComponent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "delete-component")
}

func (c *SCluster) PerformDeleteComponent(ctx context.Context, userCred mcclient.TokenCredential, query, input *api.ComponentDeleteInput) (jsonutils.JSONObject, error) {
	comp, err := c.GetComponentByType(input.Type)
	if err != nil {
		return nil, err
	}
	return nil, comp.DoDelete(ctx, userCred, input.JSON(input), "")
}

func (c *SCluster) AllowPerformUpdateComponent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "update-component")
}

func (c *SCluster) PerformUpdateComponent(ctx context.Context, userCred mcclient.TokenCredential, query, input *api.ComponentUpdateInput) (jsonutils.JSONObject, error) {
	comp, err := c.GetComponentByType(input.Type)
	if err != nil {
		return nil, err
	}
	drv, err := comp.GetDriver()
	if err != nil {
		return nil, err
	}
	if err := drv.ValidateUpdateData(input); err != nil {
		return nil, err
	}
	if err := comp.DoUpdate(ctx, userCred, input); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *SCluster) GetProjectId() string {
	return c.ProjectId
}

func (c *SCluster) prepareStartSync() error {
	if c.GetStatus() != api.ClusterStatusRunning {
		return errors.Errorf("Cluster status is %s", c.GetStatus())
	}
	return nil
}

func (m *SClusterManager) StartAutoSyncTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	clusters, err := m.GetRunningClusters()
	if err != nil {
		log.Errorf("Start auto sync cluster task get running clusters: %v", err)
		return
	}
	for _, cls := range clusters {
		cls.(*SCluster).SubmitSyncTask(ctx, userCred, nil)
	}
}

func (c *SCluster) SubmitSyncTask(ctx context.Context, userCred mcclient.TokenCredential, waitChan chan error) {
	RunSyncClusterTask(func() {
		log.Infof("start sync cluster %s", c.GetName())
		if err := c.prepareStartSync(); err != nil {
			log.Errorf("sync cluster task error: %v", err)
			if waitChan != nil {
				waitChan <- err
			}
			return
		}
		// full mode sync
		for _, man := range []IClusterModelManager{
			NamespaceManager,
			ReleaseManager,
			NodeManager,
			PodManager,
		} {
			if ret := SyncClusterResources(ctx, man, userCred, c); ret.IsError() {
				log.Errorf("Sync cluster %s resource %s error: %v", c.GetName(), man.KeywordPlural(), ret.Result())
			} else {
				log.Infof("Sync cluster %s resource %s completed: %v", c.GetName(), man.KeywordPlural(), ret.Result())
			}
		}
	})
}

func (c *SCluster) GetK8sResourceManager(kindName string) manager.IK8sResourceManager {
	return GetK8sResourceManagerByKind(kindName)
}

type sClusterUsage struct {
	Id string
}

func (m *SClusterManager) usageClusters(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) ([]sClusterUsage, error) {
	q := m.Query("id")
	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}
	var clusters []sClusterUsage
	if err := q.All(&clusters); err != nil {
		return nil, errors.Wrap(err, "query cluster usage")
	}
	return clusters, nil
}

func (m *SClusterManager) Usage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) (*api.ClusterUsage, error) {
	usage := new(api.ClusterUsage)
	clusters, err := m.usageClusters(scope, ownerId)
	if err != nil {
		return nil, err
	}
	usage.Count = int64(len(clusters))
	nodeUsage, err := NodeManager.Usage(clusters)
	if err != nil {
		return nil, errors.Wrap(err, "get node usage")
	}
	usage.Node = nodeUsage
	return usage, nil
}
