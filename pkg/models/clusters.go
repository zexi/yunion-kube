package models

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/util/wait"
	yutils "yunion.io/x/pkg/utils"
	ykecluster "yunion.io/x/yke/pkg/cluster"
	ykek8s "yunion.io/x/yke/pkg/k8s"
	yketypes "yunion.io/x/yke/pkg/types"

	"yunion.io/x/yunion-kube/pkg/clusterdriver"
	drivertypes "yunion.io/x/yunion-kube/pkg/clusterdriver/types"
	ykedriver "yunion.io/x/yunion-kube/pkg/clusterdriver/yke"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/templates"
	"yunion.io/x/yunion-kube/pkg/types/apis"
	"yunion.io/x/yunion-kube/pkg/utils"
)

var ClusterManager *SClusterManager

var (
	ClusterNotFoundError = errors.New("Cluster not found")
)

func init() {
	ClusterManager = &SClusterManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(SCluster{}, "clusters_tbl", "kube_cluster", "kube_clusters"),
	}
}

const (
	CLUSTER_STATUS_INIT       = "init"
	CLUSTER_STATUS_CREATING   = "creating"
	CLUSTER_STATUS_IMPORT     = "importing"
	CLUSTER_STATUS_POST_CHECK = "post-checking"
	CLUSTER_STATUS_RUNNING    = "running"
	CLUSTER_STATUS_ERROR      = "error"
	CLUSTER_STATUS_DEPLOY     = "deploying"
	CLUSTER_STATUS_UPDATING   = "updating"
	CLUSTER_STATUS_DELETING   = "deleting"

	CLUSTER_MODE_INTERNAL = "internal"

	DEFAULT_CLUSER_MODE              = CLUSTER_MODE_INTERNAL
	DEFAULT_CLUSER_CIDR              = "10.43.0.0/16"
	DEFAULT_SERVICE_CLUSTER_IP_RANGE = "10.43.0.0/16"
	DEFAULT_CLUSTER_DOMAIN           = "cluster.local"
	DEFAULT_INFRA_CONTAINER_IMAGE    = "yunion/pause-amd64:3.0"

	K8S_PROXY_URL_PREFIX    = "/k8s/clusters/"
	K8S_AUTH_WEBHOOK_PREFIX = "/k8s/auth/"
)

var (
	ClusterProcessingStatus = sets.NewString(
		CLUSTER_STATUS_DEPLOY,
		CLUSTER_STATUS_IMPORT,
		CLUSTER_STATUS_UPDATING,
		CLUSTER_STATUS_CREATING,
		CLUSTER_STATUS_DELETING,
	)
)

type SClusterManager struct {
	db.SStatusStandaloneResourceBaseManager
	models.SInfrastructureManager
}

type SCluster struct {
	db.SStatusStandaloneResourceBase
	models.SInfrastructure
	Mode          string `nullable:"false" create:"required" list:"user"`
	K8sVersion    string `nullable:"false" create:"required" list:"user" update:"user"`
	ClusterCidr   string `nullable:"true" create:"optional" list:"user"`
	ClusterDomain string `nullable:"true" create:"optional" list:"user"`
	//ServiceClusterIPRange string `nullable:"false" create:"optional" default:"10.43.0.0/16" list:"user"`
	InfraContainerImage string `nullable:"true" create:"optional" list:"user"`

	ApiEndpoint         string `nullable:"true" list:"user"`
	ClientCertificate   string `nullable:"true"`
	ClientKey           string `nullable:"true"`
	RootCaCertificate   string `nullable:"true"`
	ServiceAccountToken string `nullable:"true"`
	Certs               string `nullable:"true"`
	YkeConfig           string `nullable:"true"`
}

func (m *SClusterManager) InitializeData() error {
	// check if default cluster exists
	cluster, err := m.FetchByIdOrName(nil, "default")
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		defCluster := SCluster{}
		defCluster.Id = stringutils.UUID4()
		defCluster.Name = "default"
		defCluster.K8sVersion = DEFAULT_K8S_VERSION
		defCluster.Mode = DEFAULT_CLUSER_MODE
		defCluster.ClusterCidr = DEFAULT_CLUSER_CIDR
		defCluster.ClusterDomain = DEFAULT_CLUSTER_DOMAIN
		//defCluster.ServiceClusterIPRange = DEFAULT_SERVICE_CLUSTER_IP_RANGE
		defCluster.InfraContainerImage = DEFAULT_INFRA_CONTAINER_IMAGE
		defCluster.Status = CLUSTER_STATUS_INIT
		err = m.TableSpec().Insert(&defCluster)
		if err != nil {
			return fmt.Errorf("Insert default cluster error: %v", err)
		}
	} else {
		c := cluster.(*SCluster)
		if c.ClusterCidr != DEFAULT_SERVICE_CLUSTER_IP_RANGE {
			_, err = cluster.GetModelManager().TableSpec().Update(c, func() error {
				c.ClusterCidr = DEFAULT_CLUSER_CIDR
				return nil
			})
			return err
		}
	}
	return nil
}

func (m *SClusterManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (m *SClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	setDefaultStr := func(key, def string) string {
		val, _ := data.GetString(key)
		if val == "" {
			val = def
			data.Set(key, jsonutils.NewString(val))
		}
		return val
	}

	k8sVersion := setDefaultStr("k8s_version", DEFAULT_K8S_VERSION)
	_, err := K8sYKEVersionMap.GetYKEVersion(k8sVersion)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Invalid version %q: %v", k8sVersion, err)
	}

	mode := setDefaultStr("mode", DEFAULT_CLUSER_MODE)
	if !sets.NewString(CLUSTER_MODE_INTERNAL).Has(mode) {
		return nil, httperrors.NewInputParameterError("Invalid cluster mode: %q", mode)
	}

	setDefaultStr("cluster_cidr", DEFAULT_CLUSER_CIDR)
	setDefaultStr("cluster_domain", DEFAULT_CLUSTER_DOMAIN)
	setDefaultStr("infra_container_image", DEFAULT_INFRA_CONTAINER_IMAGE)
	return m.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (m *SClusterManager) FetchClusterByIdOrName(userCred mcclient.IIdentityProvider, ident string) (*SCluster, error) {
	cluster, err := m.FetchByIdOrName(userCred, ident)
	if err != nil {
		log.Errorf("Fetch cluster %q fail: %v", ident, err)
		if err == sql.ErrNoRows {
			return nil, ClusterNotFoundError
		}
		return nil, err
	}
	return cluster.(*SCluster), nil
}

func (m *SClusterManager) FetchClusterById(ident string) (*SCluster, error) {
	cluster, err := m.FetchById(ident)
	if err != nil {
		log.Errorf("Fetch cluster by id %q fail: %v", ident, err)
		if err == sql.ErrNoRows {
			return nil, ClusterNotFoundError
		}
		return nil, err
	}
	return cluster.(*SCluster), nil
}

func (m *SClusterManager) GetClusterById(ident string) (*apis.Cluster, error) {
	cluster, err := m.FetchClusterById(ident)
	if err != nil {
		return nil, err
	}
	return cluster.Cluster()
}

func (m *SClusterManager) GetDeployConfig(cluster *SCluster, pendingNodes ...*SNode) (*yketypes.KubernetesEngineConfig, error) {
	oldConf, fixedConf, err := m.getDeployConfig(cluster, pendingNodes...)
	if err != nil {
		return nil, err
	}
	if reflect.DeepEqual(oldConf, fixedConf) {
		return oldConf, nil
	}
	return fixedConf, nil
}

func (m *SClusterManager) getDeployConfig(cluster *SCluster, pendingNodes ...*SNode) (oldConf, fixedConf *yketypes.KubernetesEngineConfig, err error) {
	clusterId := cluster.Id
	oldConf, err = cluster.GetYKEConfig()
	if err != nil {
		err = fmt.Errorf("Get old YKE config: %v", err)
		return
	}

	nodes, err := m.reconcileYKENodes(clusterId, pendingNodes...)
	if err != nil {
		err = fmt.Errorf("Get YKE nodes: %v", err)
		return
	}
	fixedConf, err = cluster.NewYKEFixedConfig()
	if err != nil {
		err = fmt.Errorf("Get cluster new YKE config: %v", err)
		return
	}
	fixedConf.Nodes = nodes
	return oldConf, fixedConf, nil
}

func (m *SClusterManager) reconcileYKENodes(clusterId string, pendingNodes ...*SNode) ([]yketypes.ConfigNode, error) {
	objs, err := NodeManager.ListByCluster(clusterId)
	if err != nil {
		return nil, err
	}
	objs = mergePendingNodes(objs, pendingNodes)
	etcd := false
	controlplane := false
	var nodes []yketypes.ConfigNode
	for _, obj := range objs {
		if obj.Etcd {
			etcd = true
		}
		if obj.Controlplane {
			controlplane = true
		}

		node := obj.GetNodeConfig()
		if node.Address == "" {
			log.Warningf("Node %q may not registered, skip it", node.NodeName)
			continue
		}
		nodes = append(nodes, *node)
	}
	if !etcd || !controlplane {
		return nil, errors.New("waiting for etcd and controlplane nodes to be registered")
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeName < nodes[j].NodeName
	})
	return nodes, nil
}

func (m *SClusterManager) GetInternalClusters() ([]SCluster, error) {
	q := ClusterManager.Query()
	q = q.Equals("mode", CLUSTER_MODE_INTERNAL)

	ret := []SCluster{}
	err := q.All(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *SCluster) GetYKESystemImages() (*yketypes.SystemImages, error) {
	if c.K8sVersion == "" {
		c.K8sVersion = DEFAULT_K8S_VERSION
	}
	ykeVersion, err := K8sYKEVersionMap.GetYKEVersion(c.K8sVersion)
	if err != nil {
		return nil, err
	}
	imageDefaults := yketypes.K8sVersionToSystemImages[ykeVersion]
	return &imageDefaults, nil
}

func (c *SCluster) Cluster() (*apis.Cluster, error) {
	ykeConf, err := c.GetYKEConfig()
	if err != nil {
		return nil, err
	}

	return &apis.Cluster{
		Id:                           c.Id,
		Name:                         c.Name,
		CaCert:                       c.RootCaCertificate,
		ApiEndpoint:                  c.ApiEndpoint,
		YunionKubernetesEngineConfig: ykeConf,
	}, nil
}

func (c *SCluster) ToInfo() *drivertypes.ClusterInfo {
	info := &drivertypes.ClusterInfo{
		ClientCertificate:   c.ClientCertificate,
		ClientKey:           c.ClientKey,
		RootCaCertificate:   c.RootCaCertificate,
		ServiceAccountToken: c.ServiceAccountToken,
		Version:             c.K8sVersion,
		Endpoint:            c.ApiEndpoint,
		Config:              c.YkeConfig,
	}
	if info.RootCaCertificate != "" && info.Endpoint != "" {
		dInfo, err := DecodeClusterInfo(info)
		if err != nil {
			log.Errorf("DecodeClusterInfo error: %v", err)
			return info
		}
		config, _ := templates.GetKubeConfig(dInfo.Endpoint, c.Name, "kube-admin", dInfo.RootCaCertificate, dInfo.ClientCertificate, dInfo.ClientKey)
		info.KubeConfig = config
	}
	return info
}

func DecodeClusterInfo(info *drivertypes.ClusterInfo) (*drivertypes.ClusterInfo, error) {
	certBytes, err := base64.StdEncoding.DecodeString(info.ClientCertificate)
	if err != nil {
		return nil, err
	}
	keyBytes, err := base64.StdEncoding.DecodeString(info.ClientKey)
	if err != nil {
		return nil, err
	}
	rootBytes, err := base64.StdEncoding.DecodeString(info.RootCaCertificate)
	if err != nil {
		return nil, err
	}

	return &drivertypes.ClusterInfo{
		ClientCertificate:   string(certBytes),
		ClientKey:           string(keyBytes),
		RootCaCertificate:   string(rootBytes),
		ServiceAccountToken: info.ServiceAccountToken,
		Version:             info.Version,
		Endpoint:            info.Endpoint,
		Config:              info.Config,
	}, nil
}

func (c *SCluster) GetNodes() ([]*SNode, error) {
	return NodeManager.ListByCluster(c.Id)
}

func (c *SCluster) GetYKENodes() ([]*SNode, error) {
	ykeConf, err := c.GetYKEConfig()
	if err != nil {
		return nil, err
	}
	nodes := ykeConf.Nodes
	objs := make([]*SNode, 0)
	for _, node := range nodes {
		parts := strings.Split(node.NodeName, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("Invalid yke node name: %q", node.NodeName)
		}
		clusterId, nodeId := parts[0], parts[1]
		if clusterId != c.Id {
			return nil, fmt.Errorf("YKE node %q cluster id %s not equal '%s:%s'", nodeId, clusterId, c.Name, c.Id)
		}
		obj, err := NodeManager.FetchNodeById(nodeId)
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

func (c *SCluster) AllowPerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

func (c *SCluster) ValidateDeployCondition(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (nodes []*SNode, err error) {
	nodes, err = c.GetNodes()
	if err != nil {
		return
	}
	isForce := jsonutils.QueryBoolean(data, "force", false)
	if isForce {
		return
	}
	if c.Status == CLUSTER_STATUS_DEPLOY {
		err = httperrors.NewUnsupportOperationError("Cluster status is %q", c.Status)
		return
	}
	for _, node := range nodes {
		if !yutils.IsInStringArray(node.Status, []string{NODE_STATUS_READY, NODE_STATUS_RUNNING}) {
			err = httperrors.NewUnsupportOperationError("Node %q status %q is not ready or running", node.Name, node.Status)
			return
		}
	}
	return
}

func FetchClusterDeployTaskData(pendingNodes []*SNode) *jsonutils.JSONDict {
	ret := jsonutils.NewDict()
	ids := []string{}
	for _, node := range pendingNodes {
		ids = append(ids, node.Id)
	}
	ret.Add(jsonutils.NewStringArray(ids), NODES_DEPLOY_IDS_KEY)
	return ret
}

func (c *SCluster) PerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	nodes, err := c.ValidateDeployCondition(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	err = c.StartClusterDeployTask(ctx, userCred, FetchClusterDeployTaskData(nodes), "")
	return nil, err
}

func (c *SCluster) StartClusterDeployTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeployTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) backoffTryRun(f func() error, failureThreshold int) {
	run := func() error {
		backoff := wait.Backoff{
			Duration: 5 * time.Second,
			Factor:   2, // double the timeout for every failure
			Steps:    failureThreshold,
		}
		return wait.ExponentialBackoff(backoff, func() (bool, error) {
			err := f()
			if err != nil {
				// Retry until the timeout
				log.Errorf("backoffTryRun cluster %q error: %v", c.Name, err)
				c.SetStatus(GetAdminCred(), CLUSTER_STATUS_ERROR, err.Error())
				return false, nil
			}
			// The last f() call was a success, return cleanly
			return true, nil
		})
	}

	go run()
}

func (c *SCluster) AllowPerformSyncConfig(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

func (c *SCluster) PerformSyncConfig(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	nodes, err := c.ValidateDeployCondition(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	taskData := FetchClusterDeployTaskData(nodes)
	taskData.Add(jsonutils.JSONTrue, "sync")
	err = c.StartClusterDeployTask(ctx, userCred, taskData, "")
	return nil, err
}

func (c *SCluster) SyncUpdate(ctx context.Context) error {
	ykeConfStr := c.YkeConfig
	opts := &drivertypes.DriverOptions{
		StringOptions: map[string]string{"ykeConfig": ykeConfStr},
	}
	driver := c.ClusterDriver()
	clusterInfo := c.ToInfo()
	c.SetStatus(GetAdminCred(), CLUSTER_STATUS_UPDATING, "from sync update")
	info, err := driver.Update(context.Background(), opts, clusterInfo)
	if err != nil {
		c.SetStatus(GetAdminCred(), CLUSTER_STATUS_ERROR, err.Error())
		return err
	}
	if info != nil {
		err = c.saveClusterInfo(info)
		if err != nil {
			log.Errorf("Save cluster info after update error: %v", err)
			return err
		}
	}
	return c.SetStatus(GetAdminCred(), CLUSTER_STATUS_RUNNING, "")
}

func (c *SCluster) Deploy(ctx context.Context, callback func(), pendingNodes ...*SNode) error {
	ykeConf, err := ClusterManager.GetDeployConfig(c, pendingNodes...)
	if err != nil {
		return err
	}
	if ykeConf == nil {
		log.Warningf("ykeConf is none, config not change, skip this deploy")
		return nil
	}

	ykeConfStr, err := utils.ConvertYkeConfigToStr(*ykeConf)
	if err != nil {
		return err
	}
	opts := &drivertypes.DriverOptions{
		StringOptions: map[string]string{"ykeConfig": ykeConfStr},
	}

	driver := c.ClusterDriver()
	clusterInfo := c.ToInfo()

	deployF := func() (err error) {
		err = c.SetStatus(GetAdminCred(), CLUSTER_STATUS_DEPLOY, "")
		if err != nil {
			return
		}
		err = setNodesStatus(pendingNodes, NODE_STATUS_DEPLOY)
		if err != nil {
			return
		}

		info, err := driver.Update(context.Background(), opts, clusterInfo)
		if err != nil {
			log.Errorf("cluster update err: %v", err)
			setNodesStatus(pendingNodes, NODE_STATUS_ERROR)
			return
		}

		if info != nil {
			err = c.saveClusterInfo(info)
			if err != nil {
				log.Errorf("Save cluster info after update error: %v", err)
				return
			}
		}
		setNodesStatus(pendingNodes, NODE_STATUS_RUNNING)
		c.SetStatus(GetAdminCred(), CLUSTER_STATUS_RUNNING, "")
		if callback != nil {
			callback()
		}
		return nil
	}

	c.backoffTryRun(deployF, 1)
	return nil
}

func SetNodesStatusByJson(data *jsonutils.JSONDict, status string) error {
	nodes, err := data.GetArray(NODES_DEPLOY_IDS_KEY)
	if err != nil {
		return err
	}
	for _, nodeId := range nodes {
		nodeIdStr, _ := nodeId.GetString()
		node, err := NodeManager.FetchNodeById(nodeIdStr)
		if err != nil {
			return err
		}
		err = node.SetStatus(GetAdminCred(), status, "")
		if err != nil {
			return err
		}
	}
	return err
}

func SetNodesStatus(nodes []*SNode, status string) error {
	return setNodesStatus(nodes, status)
}

func setNodesStatus(nodes []*SNode, status string) error {
	var err error
	for _, node := range nodes {
		err = node.SetStatus(GetAdminCred(), status, "")
		if err != nil {
			return err
		}
	}
	return err
}

func (c *SCluster) saveClusterInfo(clusterInfo *drivertypes.ClusterInfo) error {
	_, err := c.GetModelManager().TableSpec().Update(c, func() error {
		c.ClientCertificate = clusterInfo.ClientCertificate
		c.ClientKey = clusterInfo.ClientKey
		c.RootCaCertificate = clusterInfo.RootCaCertificate
		//c.K8sVersion = clusterInfo.Version
		c.ApiEndpoint = clusterInfo.Endpoint
		c.YkeConfig = clusterInfo.Config
		return nil
	})
	return err
}

func (c *SCluster) SetYKEConfig(config *yketypes.KubernetesEngineConfig) error {
	if config == nil {
		return nil
	}
	confStr, err := utils.ConvertYkeConfigToStr(*config)
	if err != nil {
		return err
	}
	_, err = c.GetModelManager().TableSpec().Update(c, func() error {
		c.YkeConfig = confStr
		return nil
	})
	return err
}

func (c *SCluster) GetYKEConfig() (conf *yketypes.KubernetesEngineConfig, err error) {
	confStr := c.YkeConfig
	if confStr == "" {
		return
	}
	return utils.ConvertToYkeConfig(confStr)
}

func (c *SCluster) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("Cluster delete do nothing")
	return nil
}

func (c *SCluster) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return c.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (c *SCluster) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	nodes, err := NodeManager.ListByCluster(c.Id)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		log.Debugf("No nodes belongs to cluster %q", c.Name)
		return c.RealDelete(ctx, userCred)
	}
	c.SetStatus(GetAdminCred(), CLUSTER_STATUS_DELETING, "")
	return c.startRemoveCluster(ctx, userCred)
}

func (c *SCluster) startRemoveCluster(ctx context.Context, userCred mcclient.TokenCredential) error {
	deleteF := func() (err error) {
		log.Infof("Deleting cluster [%s]", c.Name)
		err = c.RemoveCluster(ctx)
		if err != nil {
			log.Errorf("Delete cluster error: %v", err)
			return
		}
		return
	}

	c.backoffTryRun(deleteF, 3)
	return nil
}

func (c *SCluster) RemoveCluster(ctx context.Context) error {
	err := c.ClusterDriver().Remove(ctx, c.ToInfo())
	if err != nil {
		return fmt.Errorf("Remove cluster error: %v", err)
	}

	if c.Name == "default" {
		// clear cluster info
		c.saveClusterInfo(&drivertypes.ClusterInfo{})
		c.SetStatus(GetAdminCred(), CLUSTER_STATUS_INIT, "")
		return nil
	}
	return c.RealDelete(ctx, GetAdminCred())
}

func (c *SCluster) ClusterDriver() drivertypes.Driver {
	driver := clusterdriver.Drivers["yke"]
	return driver
}

func (c *SCluster) ClusterYKEDriver() *ykedriver.Driver {
	driver := clusterdriver.Drivers["yke"]
	return driver.(*ykedriver.Driver)
}

func (c *SCluster) GetYKEAuthzConfig() yketypes.AuthzConfig {
	return yketypes.AuthzConfig{Mode: ykecluster.DefaultAuthorizationMode}
}

func (c *SCluster) GetCloudProviderConfig(conf yketypes.YunionConfig) yketypes.CloudProvider {
	return yketypes.CloudProvider{
		Name: "yunion",
		YunionCloudProvider: &yketypes.YunionCloudProvider{
			AuthURL:       conf.AuthURL,
			AdminUser:     conf.AdminUser,
			AdminPassword: conf.AdminPassword,
			AdminProject:  conf.AdminProject,
			Region:        conf.Region,
			Cluster:       c.Name,
		},
	}
}

func (c *SCluster) GetYunionConfig() yketypes.YunionConfig {
	session, err := GetAdminSession()
	if err != nil {
		log.Errorf("Get admin session error: %v", err)
	}
	influxdbUrl, err := session.GetServiceURL("influxdb", "internalURL")
	if err != nil {
		log.Errorf("Get internal influxdb endpoint error: %v", err)
	}
	schedulerUrl, err := session.GetServiceURL("scheduler", "internalURL")
	if err != nil {
		log.Errorf("Get internal scheduler endpoint error: %v", err)
	}
	o := options.Options
	return yketypes.YunionConfig{
		AuthURL:        o.AuthURL,
		AdminUser:      o.AdminUser,
		AdminPassword:  o.AdminPassword,
		AdminProject:   o.AdminProject,
		Region:         o.Region,
		KubeCluster:    c.Name,
		HostBridge:     "br0",
		InfluxdbUrl:    influxdbUrl,
		SchedulerUrl:   fmt.Sprintf("%s/k8s", schedulerUrl),
		DockerGraphDir: DEFAULT_DOCKER_GRAPH_DIR,
	}
}

func (c *SCluster) GetYKENetworkConfig() yketypes.NetworkConfig {
	conf := yketypes.NetworkConfig{
		Plugin: ykecluster.DefaultNetworkPlugin,
	}

	// TODO:
	// - fix this hard code bridge name
	// - not get auth info from options?
	o := options.Options
	conf.Options = map[string]string{
		ykecluster.YunionBridge:       "br0",
		ykecluster.YunionAuthURL:      o.AuthURL,
		ykecluster.YunionAdminUser:    o.AdminUser,
		ykecluster.YunionAdminPasswd:  o.AdminPassword,
		ykecluster.YunionAdminProject: o.AdminProject,
		ykecluster.YunionRegion:       o.Region,
		ykecluster.YunionKubeCluster:  c.Name,
	}
	return conf
}

func GetKubeServerUrl() (string, error) {
	session, err := GetAdminSession()
	if err != nil {
		return "", err
	}
	endpoint, err := session.GetServiceURL(KUBE_SERVER_SERVICE, INTERNAL_ENDPOINT_TYPE)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("Parse url %q error: %v", endpoint, err)
	}
	u.Path = ""
	return u.String(), nil
}

func (c *SCluster) GetK8sRestConfig() (*rest.Config, error) {
	if c.ApiEndpoint == "" {
		return nil, fmt.Errorf("cluster %q not found k8s api endpoint", c.Name)
	}
	driver := c.ClusterDriver()
	if driver == nil {
		return nil, fmt.Errorf("Cluster driver not init?")
	}
	config, err := driver.GetK8sRestConfig(c.ToInfo())
	if err != nil {
		return nil, err
	}
	// clean WrapTransport
	config.WrapTransport = nil
	return config, nil
}

func (c *SCluster) GetK8sClient() (*kubernetes.Clientset, error) {
	config, err := c.GetK8sRestConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (c *SCluster) GetK8sWebhookAuthUrl() (string, error) {
	serverUrl, err := GetKubeServerUrl()
	if err != nil {
		return "", err
	}
	parts, err := url.Parse(serverUrl)
	if err != nil {
		return "", err
	}
	parts.Path = fmt.Sprintf("%s%s", K8S_AUTH_WEBHOOK_PREFIX, c.Id)
	return parts.String(), nil
}

func (c *SCluster) GetYKEWebhookAuthConfig() (yketypes.WebhookAuth, error) {
	ret := yketypes.WebhookAuth{}
	webhookUrl, err := c.GetK8sWebhookAuthUrl()
	if err != nil {
		return ret, err
	}
	ret.URL = webhookUrl
	return ret, nil
}

func (c *SCluster) GetClusterCIDR() string {
	cidr := c.ClusterCidr
	if len(cidr) == 0 {
		return DEFAULT_CLUSER_CIDR
	}
	return cidr
}

func (c *SCluster) GetServiceClusterIPRange() string {
	//return c.ServiceClusterIPRange
	return DEFAULT_SERVICE_CLUSTER_IP_RANGE
}

func (c *SCluster) GetClusterDomain() string {
	domain := c.ClusterDomain
	if len(domain) == 0 {
		return DEFAULT_CLUSTER_DOMAIN
	}
	return domain
}

func (c *SCluster) GetClusterDNSServiceIp() string {
	// 10.43.0.0/16 to 10.43.0.10
	network := strings.Split(c.GetClusterCIDR(), "/")[0]
	segs := strings.Split(network, ".")[:3]
	segs = append(segs, "10")
	return strings.Join(segs, ".")
}

func (c *SCluster) GetYKEServicesConfig(images yketypes.SystemImages) (yketypes.ConfigServices, error) {
	config := yketypes.ConfigServices{}
	config.Etcd = yketypes.ETCDService{
		BaseService: yketypes.BaseService{Image: images.Etcd},
	}

	config.KubeAPI = yketypes.KubeAPIService{
		BaseService: yketypes.BaseService{
			Image: images.Kubernetes,
			ExtraArgs: map[string]string{
				"authentication-token-webhook-config-file": "/etc/kubernetes/webhook.kubeconfig",
				"feature-gates": "CSIPersistentVolume=true,MountPropagation=true",
				//"feature-gates":  "CSIPersistentVolume=true,MountPropagation=true,KubeletPluginsWatcher=true,CSINodeInfo=true,CSIDriverRegistry=true",
				"runtime-config": "storage.k8s.io/v1alpha1=true,admissionregistration.k8s.io/v1alpha1=true,settings.k8s.io/v1alpha1=true",
			},
		},
		PodSecurityPolicy:     false,
		ServiceClusterIPRange: c.GetServiceClusterIPRange(),
	}

	config.KubeController = yketypes.KubeControllerService{
		BaseService: yketypes.BaseService{
			Image: images.Kubernetes,
			ExtraArgs: map[string]string{
				"feature-gates":  "CSIPersistentVolume=true,MountPropagation=true,VolumeScheduling=true",
				"cloud-provider": "external",
			},
		},
		ServiceClusterIPRange: c.GetServiceClusterIPRange(),
		ClusterCIDR:           c.GetClusterCIDR(),
	}

	config.Scheduler = yketypes.SchedulerService{
		BaseService: yketypes.BaseService{
			Image: images.Kubernetes,
			ExtraArgs: map[string]string{
				"feature-gates":      "CSIPersistentVolume=true,MountPropagation=true,VolumeScheduling=true",
				"policy-config-file": ykecluster.SchedulerConfigPath,
			},
		},
	}

	infraContainerImage := c.InfraContainerImage
	if len(infraContainerImage) == 0 {
		infraContainerImage = images.PodInfraContainer
	}

	config.Kubelet = yketypes.KubeletService{
		BaseService: yketypes.BaseService{
			Image: images.Kubernetes,
			ExtraArgs: map[string]string{
				"read-only-port": "10255",
				"feature-gates":  "CSIPersistentVolume=true,MountPropagation=true,KubeletPluginsWatcher=true,VolumeScheduling=true",
				"eviction-hard":  "memory.available<100Mi,nodefs.available<2Gi,nodefs.inodesFree<5%",
			},
		},
		ClusterDomain:       c.GetClusterDomain(),
		ClusterDNSServer:    c.GetClusterDNSServiceIp(),
		InfraContainerImage: infraContainerImage,
	}

	config.Kubeproxy = yketypes.KubeproxyService{
		BaseService: yketypes.BaseService{Image: images.Kubernetes},
	}

	return config, nil
}

func (c *SCluster) NewYKEFixedConfig() (*yketypes.KubernetesEngineConfig, error) {
	systemImages, err := c.GetYKESystemImages()
	if err != nil {
		return nil, err
	}
	conf := yketypes.KubernetesEngineConfig{}
	conf.SystemImages = *systemImages
	conf.Authorization = c.GetYKEAuthzConfig()
	conf.Network = c.GetYKENetworkConfig()
	conf.YunionConfig = c.GetYunionConfig()
	conf.CloudProvider = c.GetCloudProviderConfig(conf.YunionConfig)
	conf.Services, err = c.GetYKEServicesConfig(conf.SystemImages)
	if err != nil {
		return nil, err
	}
	conf.WebhookAuth, err = c.GetYKEWebhookAuthConfig()
	return &conf, err
}

func (c *SCluster) AllowPerformGenerateKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	//return allowPerformAction(ctx, userCred, query, data)
	return true
}

func (c *SCluster) PerformGenerateKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var conf string
	var err error
	if userCred.IsSystemAdmin() {
		//directly := jsonutils.QueryBoolean(data, "directly", false)
		//getF := c.GetAdminProxyKubeConfig
		//if directly {
		getF := c.GetAdminKubeconfig
		//}
		conf, err = getF()
	} else {
		conf, err = c.GetClientTokenKubeConfig(userCred)
	}
	if err != nil {
		return nil, httperrors.NewInternalServerError("Generate kubeconfig err: %v", err)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(conf), "kubeconfig")
	return ret, nil
}

func (c *SCluster) GetClientTokenKubeConfig(userCred mcclient.TokenCredential) (string, error) {
	info, err := DecodeClusterInfo(c.ToInfo())
	if err != nil {
		return "", err
	}
	project := userCred.GetProjectName()
	token := userCred.GetTokenString()
	expired := userCred.GetExpires()
	return templates.GetKubeTokenConfig(info.Endpoint, c.Name, project, project, token, expired)
}

func (c *SCluster) GetAdminProxyKubeConfig() (string, error) {
	info, err := DecodeClusterInfo(c.ToInfo())
	if err != nil {
		return "", err
	}
	endpoint, err := c.ClusterProxyEndpoint()
	if err != nil {
		return "", err
	}
	return templates.GetKubeConfigByProxy(endpoint, c.Name, "kube-client", info.RootCaCertificate, info.ClientCertificate, info.ClientKey)
}

func (c *SCluster) ClusterProxyEndpoint() (string, error) {
	serverUrl, err := GetKubeServerUrl()
	if err != nil {
		return "", err
	}
	parts, err := url.Parse(serverUrl)
	if err != nil {
		return "", err
	}
	parts.Path = fmt.Sprintf("%s%s", K8S_PROXY_URL_PREFIX, c.Id)
	return parts.String(), nil
}

func (c *SCluster) GetAdminKubeconfig() (string, error) {
	info, err := DecodeClusterInfo(c.ToInfo())
	if err != nil {
		return "", err
	}
	return templates.GetKubeConfig(info.Endpoint, c.Name, "kube-admin", info.RootCaCertificate, info.ClientCertificate, info.ClientKey)
}

func (c *SCluster) AllowGetDetailsEngineConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.AllowGetDetails(ctx, userCred, query)
}

func (c *SCluster) GetDetailsEngineConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	configStr := c.YkeConfig
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(configStr), "config")
	return ret, nil
}

func (c *SCluster) AllowPerformUpdateEngineConfig(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

func (c *SCluster) PerformUpdateEngineConfig(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	configStr, err := data.GetString("config")
	if err != nil {
		return nil, err
	}
	config, err := utils.ConvertToYkeConfig(configStr)
	if err != nil {
		return nil, httperrors.NewBadRequestError("Invalid engine config string: %s", configStr)
	}
	nodes, err := c.GetNodes()
	if err != nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("Get nodes error: %v", err))
	}
	for _, node := range nodes {
		err = node.UpdateRolesByConfig(config)
		if err != nil {
			return nil, err
		}
	}
	err = c.SetYKEConfig(config)
	if err != nil {
		return nil, err
	}
	return c.GetDetailsEngineConfig(ctx, userCred, query)
}

func (c *SCluster) AllowGetDetailsWebhookAuthUrl(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.AllowGetDetails(ctx, userCred, query)
}

func (c *SCluster) GetDetailsWebhookAuthUrl(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	webhookUrl, err := c.GetK8sWebhookAuthUrl()
	if err != nil {
		return nil, httperrors.NewInternalServerError(err.Error())
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(webhookUrl), "url")
	return ret, nil
}

func (c *SCluster) AllowGetDetailsCloudHosts(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.AllowGetDetails(ctx, userCred, query)
}

func (c *SCluster) GetDetailsCloudHosts(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	session, err := GetAdminSession()
	if err != nil {
		err = httperrors.NewInternalServerError("Get admin session: %v", err)
		return nil, err
	}
	params := jsonutils.NewDict()
	filter := jsonutils.NewArray()
	filter.Add(jsonutils.NewString(fmt.Sprintf("host_type.in(%s, %s)", "hypervisor", "kubelet")))
	filter.Add(jsonutils.NewString("host_status.equals(online)"))
	filter.Add(jsonutils.NewString("status.equals(running)"))
	params.Add(filter, "filter")
	result, err := cloudmod.Hosts.List(session, params)
	if err != nil {
		return nil, err
	}
	hosts := jsonutils.NewArray()
	canDeploy := jsonutils.QueryBoolean(query, "can_deploy", false)
	for _, host := range result.Data {
		id, _ := host.GetString("id")
		if len(id) == 0 {
			continue
		}
		node, _ := NodeManager.FetchNodeByHostId(id)
		if node != nil && canDeploy {
			continue
		}
		hosts.Add(host)
	}
	return hosts, nil
}

func (c *SCluster) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	k8sVersion, _ := data.GetString("k8s_version")
	if k8sVersion != "" {
		_, err := K8sYKEVersionMap.GetYKEVersion(k8sVersion)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Invalid version %q: %v", k8sVersion, err)
		}
	}
	return c.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (c *SCluster) AllowPerformImport(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

type clusterImportValidator struct {
	ctx      context.Context
	cluster  *SCluster
	userCred mcclient.TokenCredential
	query    jsonutils.JSONObject
	data     jsonutils.JSONObject
}

func (v *clusterImportValidator) clusterValidate() error {
	nodes, err := v.cluster.GetNodes()
	if err != nil {
		return httperrors.NewInternalServerError("Get nodes by cluster %q: %v", v.cluster.Id, err)
	}
	if len(nodes) != 0 {
		return httperrors.NewInputParameterError("Not empty cluster %q has %d nodes", v.cluster.Name, len(nodes))
	}
	return nil
}

func (v *clusterImportValidator) ykeValidate(k8sCli *kubernetes.Clientset) (ykeConf *yketypes.KubernetesEngineConfig, err error) {
	stateConfigMap, err := ykek8s.GetConfigMap(k8sCli, ykecluster.StateConfigMapName)
	if err != nil {
		err = httperrors.NewInputParameterError("Get YKE config map from k8s cluster error: %v", err)
		return
	}
	ykeConfStr, ok := stateConfigMap.Data[ykecluster.StateConfigMapName]
	if !ok {
		err = httperrors.NewInputParameterError("Not found %q data from YKE configmap %#v", ykecluster.StateConfigMapName, stateConfigMap.Data)
		return
	}
	ykeConf, err = utils.ConvertToYkeConfig(ykeConfStr)
	if err != nil {
		err = httperrors.NewInputParameterError("Convert to YKE config error: %v", err)
		return
	}
	if len(ykeConf.Nodes) == 0 {
		err = httperrors.NewInputParameterError("YKE config nodes is empty, your config is: %q", ykeConfStr)
	}
	return
}

func (v *clusterImportValidator) kubeConfigValidate() (k8sConf *rest.Config, err error) {
	kubeConfigStr, err := v.data.GetString("kube_config")
	if err != nil {
		err = httperrors.NewInputParameterError("Not found kube_config: %v", err)
		return
	}

	k8sConf, err = utils.GetK8sRestConfigFromBytes([]byte(kubeConfigStr))
	if err != nil {
		err = httperrors.NewInputParameterError("Convert kube config to rest config error: %v", err)
		return
	}
	if len(k8sConf.Host) == 0 {
		err = httperrors.NewInputParameterError("Not found kubernetes api server host")
		return
	}
	if len(k8sConf.CAData) == 0 {
		err = httperrors.NewInputParameterError("Kubeconfig 'certificate-authority-data' must provide")
		return
	}
	if len(k8sConf.CertData) == 0 {
		err = httperrors.NewInputParameterError("Kubeconfig 'client-certificate-data' must provide")
		return
	}
	if len(k8sConf.KeyData) == 0 {
		err = httperrors.NewInputParameterError("Kubeconfig 'client-key-data' must provide")
		return
	}
	return
}

func (v *clusterImportValidator) nodesValidate(ykeConf *yketypes.KubernetesEngineConfig) (nodes []*YkeConfigNodeFactory, err error) {
	nodes = make([]*YkeConfigNodeFactory, 0)
	for _, node := range ykeConf.Nodes {
		var nodeF *YkeConfigNodeFactory
		nodeF, err = NewYkeConfigNodeFactory(v.ctx, v.userCred, node, v.cluster)
		if err != nil {
			return
		}
		nodes = append(nodes, nodeF)
	}
	return
}

func (v *clusterImportValidator) do() (
	ykeConf *yketypes.KubernetesEngineConfig,
	k8sConf *rest.Config,
	nodes []*YkeConfigNodeFactory,
	err error,
) {
	err = v.clusterValidate()
	if err != nil {
		return
	}
	k8sConf, err = v.kubeConfigValidate()
	if err != nil {
		return
	}
	k8sCli, err := kubernetes.NewForConfig(k8sConf)
	if err != nil {
		return
	}
	ykeConf, err = v.ykeValidate(k8sCli)
	if err != nil {
		return
	}
	nodes, err = v.nodesValidate(ykeConf)
	return
}

type ykeImportInfo struct {
	// info from YKE config
	Version        string
	Mode           string
	CIDR           string
	ServiceIPRange string
	Domain         string
	InfraImage     string

	// info from Kube config
	ApiEndpoint       string
	RootCaCertificate string
	ClientCertificate string
	ClientKey         string
}

func getClusterYkeImportInfo(ykeConf *yketypes.KubernetesEngineConfig, kubeConf *rest.Config) (info ykeImportInfo, err error) {
	// get version from YKE config
	k8sImage := ykeConf.Services.KubeAPI.Image
	imageVersion := strings.Split(k8sImage, ":")[1]
	version, ok := YKEK8sVersionMap[imageVersion]
	if !ok {
		version = DEFAULT_K8S_VERSION
		log.Warningf("Not found support k8s image version: %q, use: %q", imageVersion, version)
	}
	cidr := ykeConf.Services.KubeController.ClusterCIDR
	if cidr == "" {
		cidr = DEFAULT_CLUSER_CIDR
	}
	//serviceCIDR := ykeConf.Services.KubeAPI.ServiceClusterIPRange
	//if serviceCIDR == "" {
	serviceCIDR := DEFAULT_SERVICE_CLUSTER_IP_RANGE
	//}
	domain := ykeConf.Services.Kubelet.ClusterDomain
	if domain == "" {
		domain = DEFAULT_CLUSTER_DOMAIN
	}
	infraImage := ykeConf.SystemImages.PodInfraContainer
	if infraImage == "" {
		infraImage = DEFAULT_INFRA_CONTAINER_IMAGE
	}

	info = ykeImportInfo{
		Version:           version,
		Mode:              DEFAULT_CLUSER_MODE,
		CIDR:              cidr,
		ServiceIPRange:    serviceCIDR,
		Domain:            domain,
		InfraImage:        infraImage,
		ApiEndpoint:       kubeConf.Host,
		RootCaCertificate: string(kubeConf.CAData),
		ClientCertificate: string(kubeConf.CertData),
		ClientKey:         string(kubeConf.KeyData),
	}
	return
}

func setClusterInfoFromImport(c *SCluster, ykeConf *yketypes.KubernetesEngineConfig, kubeConf *rest.Config) error {
	info, err := getClusterYkeImportInfo(ykeConf, kubeConf)
	if err != nil {
		return httperrors.NewInputParameterError(err.Error())
	}
	_, err = c.GetModelManager().TableSpec().Update(c, func() error {
		c.Mode = info.Mode
		c.K8sVersion = info.Version
		c.ClusterCidr = info.CIDR
		//c.ServiceClusterIPRange = info.ServiceIPRange
		c.ClusterDomain = info.Domain
		c.InfraContainerImage = info.InfraImage
		c.ApiEndpoint = info.ApiEndpoint
		c.RootCaCertificate = base64.StdEncoding.EncodeToString([]byte(info.RootCaCertificate))
		c.ClientCertificate = base64.StdEncoding.EncodeToString([]byte(info.ClientCertificate))
		c.ClientKey = base64.StdEncoding.EncodeToString([]byte(info.ClientKey))
		return nil
	})
	return err
}

func (c *SCluster) PerformImport(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	validator := &clusterImportValidator{
		ctx:      ctx,
		cluster:  c,
		userCred: userCred,
		query:    query,
		data:     data,
	}
	ykeConf, kubeConf, nodeFs, err := validator.do()
	if err != nil {
		return nil, err
	}
	err = setClusterInfoFromImport(c, ykeConf, kubeConf)
	if err != nil {
		return nil, err
	}
	for _, nodeF := range nodeFs {
		err = nodeF.Save()
		if err != nil {
			return nil, fmt.Errorf("Save node %q error: %v", nodeF.node.Name, err)
		}
	}
	c.StartClusterImportTask(ctx, userCred, fetchClusterImportTaskData(ykeConf, kubeConf, nodeFs), "")
	return nil, nil
}

func fetchClusterImportTaskData(ykeConf *yketypes.KubernetesEngineConfig, k8sConf *rest.Config, nodes []*YkeConfigNodeFactory) *jsonutils.JSONDict {
	retData := jsonutils.NewDict()
	nodesConfig := jsonutils.NewDict()
	for _, node := range nodes {
		nodesConfig.Add(node.CreateData, node.Node().Id)
	}
	retData.Add(nodesConfig, NODES_CONFIG_DATA_KEY)
	return retData
}

func (c *SCluster) StartClusterImportTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterImportTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) IsNodeAgentReady(node *SNode) bool {
	driver := c.ClusterYKEDriver()
	dialer := driver.GetDialerFactory()
	_, err := dialer.DockerDialer(c.Id, node.Id)
	if err != nil {
		log.Warningf("Node %s kube agent is not ready: %v", node.Name, err)
		return false
	}
	return true
}

func (c *SCluster) IsNodeAgentsReady(nodes ...*SNode) bool {
	isAllReady := true
	for _, node := range nodes {
		if !c.IsNodeAgentReady(node) {
			return false
		}
	}
	return isAllReady
}

func (c *SCluster) AllowPerformAddNodes(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

func (c *SCluster) validateAddNodes(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (nodes []*SNode, err error) {
	if ClusterProcessingStatus.Has(c.Status) {
		return nil, httperrors.NewNotAcceptableError(fmt.Sprintf("cluster status is %s", c.Status))
	}
	nodesData, err := data.Get("nodes")
	if err != nil {
		err = httperrors.NewInputParameterError("Found nodes data: %v", err)
		return
	}
	opts := []apis.NodeAddOption{}
	err = nodesData.Unmarshal(&opts)
	if err != nil {
		err = httperrors.NewInputParameterError("Invalid nodes data: %s", nodesData)
		return
	}
	if len(opts) == 0 {
		err = httperrors.NewInputParameterError("Empty nodes to add")
		return
	}
	for i, opt := range opts {
		opt.Cluster = c.Id
		opts[i] = opt
	}
	for _, opt := range opts {
		var node *SNode
		newData := jsonutils.Marshal(opt).(*jsonutils.JSONDict)
		node, err = NewNode(ctx, userCred, newData)
		if err != nil {
			return
		}
		nodes = append(nodes, node)
	}
	return
}

func (c *SCluster) PerformAddNodes(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	nodes, err := c.validateAddNodes(ctx, userCred, data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		err = NodeManager.TableSpec().Insert(node)
		if err != nil {
			return nil, err
		}
	}
	autoDeploy := jsonutils.QueryBoolean(data, "auto_deploy", false)
	if !autoDeploy {
		return nil, nil
	}
	c.StartClusterDeployTask(ctx, userCred, FetchClusterDeployTaskData(nodes), "")
	return nil, nil
}

func (c *SCluster) AllowPerformDeleteNodes(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

func (c *SCluster) validateDeleteNodes(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) ([]*SNode, error) {
	if ClusterProcessingStatus.Has(c.Status) {
		return nil, httperrors.NewNotAcceptableError(fmt.Sprintf("cluster status is %s", c.Status))
	}
	nodesData, err := data.GetArray("nodes")
	if err != nil {
		return nil, httperrors.NewInputParameterError("Found nodes data: %v", err)
	}
	nodes := []*SNode{}
	for _, obj := range nodesData {
		id, err := obj.GetString()
		if err != nil {
			if err != nil {
				return nil, err
			}
		}
		nodeObj, err := NodeManager.FetchByIdOrName(userCred, id)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Not found node by id: %s", id)
		}
		nodes = append(nodes, nodeObj.(*SNode))
	}

	if len(nodes) == 0 {
		return nil, httperrors.NewInputParameterError("Empty nodes id provided")
	}

	oldNodes, err := c.GetNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if len(node.Address) != 0 && strings.Contains(c.ApiEndpoint, node.Address) && len(nodes) != len(oldNodes) {
			return nil, httperrors.NewInputParameterError("First control node %q must deleted at last, address %q", node.Name, node.Address)
		}
	}
	return nodes, nil
}

func (c *SCluster) PerformDeleteNodes(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	nodes, err := c.validateDeleteNodes(ctx, userCred, data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}
	c.StartClusterDeleteNodesTask(ctx, userCred, FetchClusterDeployTaskData(nodes), "")
	return nil, nil
}

func (c *SCluster) StartClusterDeleteNodesTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	err := SetNodesStatusByJson(data, NODE_STATUS_DELETING)
	if err != nil {
		return err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeleteNodesTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) getBatchTaskNodes(nodes []*SNode) []db.IStandaloneModel {
	ret := make([]db.IStandaloneModel, len(nodes))
	for i, node := range nodes {
		ret[i] = node
	}
	return ret
}

func (c *SCluster) startClusterNodesAgentTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict,
	action, parentTaskId string,
	nodes ...*SNode,
) error {
	if data == nil {
		data = jsonutils.NewDict()
	}
	if !yutils.IsInStringArray(action, []string{"start", "stop", "restart"}) {
		return fmt.Errorf("Unspported action: %s", action)
	}
	data.Add(jsonutils.NewString(action), "action")

	objs := c.getBatchTaskNodes(nodes)
	RunBatchTask(ctx, objs, userCred, data, "ClusterNodesAgentTask", parentTaskId)
	return nil
}

func (c *SCluster) StartClusterStartNodesAgentTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict,
	parentTaskId string,
	nodes ...*SNode,
) error {
	return c.startClusterNodesAgentTask(ctx, userCred, data, "start", parentTaskId, nodes...)
}

func (c *SCluster) StartClusterRestartNodesAgentTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict,
	parentTaskId string,
	nodes ...*SNode,
) error {
	return c.startClusterNodesAgentTask(ctx, userCred, data, "restart", parentTaskId, nodes...)
}

func (c *SCluster) StartClusterStopNodesAgentTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict,
	parentTaskId string,
	nodes ...*SNode,
) error {
	return c.startClusterNodesAgentTask(ctx, userCred, data, "stop", parentTaskId, nodes...)
}

func (c *SCluster) AllowPerformRestartAgent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

func (c *SCluster) PerformRestartAgent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if ClusterProcessingStatus.Has(c.Status) {
		return nil, httperrors.NewNotAcceptableError(fmt.Sprintf("cluster status is %s", c.Status))
	}
	all, _ := data.Bool("all")
	nodes, err := c.GetNodes()
	if err != nil {
		return nil, err
	}
	targets := make([]*SNode, 0)
	if !all {
		nodeIds, _ := data.GetArray("nodes")
		if len(nodeIds) == 0 {
			return nil, httperrors.NewNotAcceptableError(fmt.Sprintf("No node provided"))
		}
		for _, nodeId := range nodeIds {
			id, err := nodeId.GetString()
			if err != nil {
				return nil, httperrors.NewNotAcceptableError(fmt.Sprintf("Get node id error: %v", err))
			}
			if node := getNodesById(nodes, id); node != nil {
				targets = append(targets, node)
			}
		}
	} else {
		targets = nodes
	}
	err = c.StartClusterRestartNodesAgentTask(ctx, userCred, nil, "", targets...)
	return nil, err
}
