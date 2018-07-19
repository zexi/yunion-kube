package models

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	ykecluster "yunion.io/yke/pkg/cluster"
	"yunion.io/yke/pkg/pki"
	yketypes "yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/httperrors"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/mcclient"
	//"yunion.io/yunioncloud/pkg/sqlchemy"
	"yunion.io/yunioncloud/pkg/util/sets"
	"yunion.io/yunioncloud/pkg/util/wait"

	"yunion.io/yunion-kube/pkg/clusterdriver"
	drivertypes "yunion.io/yunion-kube/pkg/clusterdriver/types"
	"yunion.io/yunion-kube/pkg/options"
	"yunion.io/yunion-kube/pkg/types/apis"
	"yunion.io/yunion-kube/pkg/types/slice"
	"yunion.io/yunion-kube/pkg/utils"
)

var ClusterManager *SClusterManager

var (
	ClusterNotFoundError = errors.New("Cluster not found")
)

func init() {
	ClusterManager = &SClusterManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SCluster{}, "clusters_tbl", "cluster", "clusters"),
	}
}

const (
	CLUSTER_STATUS_INIT         = "init"
	CLUSTER_STATUS_PRE_CREATING = "pre-creating"
	CLUSTER_STATUS_CREATING     = "creating"
	CLUSTER_STATUS_POST_CHECK   = "post-checking"
	CLUSTER_STATUS_RUNNING      = "running"
	CLUSTER_STATUS_ERROR        = "error"
	CLUSTER_STATUS_UPDATING     = "updating"
	CLUSTER_STATUS_DELETING     = "deleting"

	CLUSTER_MODE_INTERNAL = "internal"

	DEFAULT_CLUSER_MODE           = CLUSTER_MODE_INTERNAL
	DEFAULT_CLUSER_CIDR           = "10.43.0.0/16"
	DEFAULT_CLUSTER_DOMAIN        = "cluster.local"
	DEFAULT_INFRA_CONTAINER_IMAGE = "yunion/pause-amd64:3.0"
)

type SClusterManager struct {
	db.SVirtualResourceBaseManager
}

func (m *SClusterManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return m.SVirtualResourceBaseManager.AllowListItems(ctx, userCred, query)
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
	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (m *SClusterManager) FetchClusterByIdOrName(ownerProjId, ident string) (*SCluster, error) {
	cluster, err := m.FetchByIdOrName(ownerProjId, ident)
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

func (m *SClusterManager) AddClusterNodes(clusterId string, pendingNodes ...*SNode) error {
	cluster, err := m.FetchClusterById(clusterId)
	if err != nil {

	}
	if cluster == nil {
		return ClusterNotFoundError
	}

	if slice.ContainsString([]string{CLUSTER_STATUS_UPDATING, CLUSTER_STATUS_POST_CHECK}, cluster.Status) {
		return fmt.Errorf("Cluster %q add node: status is %q", cluster.Name, cluster.Status)
	}

	return cluster.addNodes(context.Background(), pendingNodes...)
}

func (m *SClusterManager) UpdateCluster(clusterId string, pendingNodes ...*SNode) error {
	cluster, err := m.FetchClusterById(clusterId)
	if err != nil {
		return err
	}
	if slice.ContainsString([]string{CLUSTER_STATUS_UPDATING, CLUSTER_STATUS_POST_CHECK}, cluster.Status) {
		return fmt.Errorf("Cluster %q update: status is %s", cluster.Name, cluster.Status)
	}

	return cluster.Update(context.Background(), pendingNodes...)
}

func (m *SClusterManager) GetConfig(cluster *SCluster, pendingNodes ...*SNode) (*yketypes.KubernetesEngineConfig, error) {
	oldConf, _, err := m.getConfig(false, cluster, pendingNodes...)
	if err != nil {
		return nil, err
	}
	_, newConf, err := m.getConfig(true, cluster, pendingNodes...)
	if err != nil {
		return nil, err
	}
	if reflect.DeepEqual(oldConf, newConf) {
		newConf = nil
		log.Debugf("config not change")
	}
	return newConf, nil
}

func (m *SClusterManager) getConfig(reconcileYKE bool, cluster *SCluster, pendingNodes ...*SNode) (old, new *yketypes.KubernetesEngineConfig, err error) {
	clusterId := cluster.Id
	old, err = cluster.GetYKEConfig()
	if err != nil {
		return nil, nil, err
	}
	if reconcileYKE {
		nodes, err := m.reconcileYKENodes(clusterId, pendingNodes...)
		if err != nil {
			return nil, nil, err
		}
		newConf, err := cluster.NewYKEConfig()
		if err != nil {
			return nil, nil, err
		}
		if old != nil {
			newConf = *old
		}
		newConf.Nodes = nodes
		new = &newConf
	}
	return old, new, nil
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

type SCluster struct {
	db.SVirtualResourceBase
	Mode          string `nullable:"false" create:"required" list:"user"`
	K8sVersion    string `nullable:"false" create:"required" list:"user"`
	ClusterCidr   string `nullable:"true" create:"optional" list:"user"`
	ClusterDomain string `nullable:"true" create:"optional" list:"user"`
	//ServiceClusterIPRange string `nullable:"false" create:"optional" default:"10.43.0.0/16" list:"user"`
	InfraContainerImage string `nullable:"true" create:"optional" list:"user"`

	ApiEndpoint         string               `nullable:"true" list:"user"`
	ClientCertificate   string               `nullable:"true"`
	ClientKey           string               `nullable:"true"`
	RootCaCertificate   string               `nullable:"true"`
	ServiceAccountToken string               `nullable:"true"`
	Certs               string               `nullable:"true"`
	YkeConfig           string               `nullable:"true"`
	Metadata            jsonutils.JSONObject `nullable:"true"`
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
	metadata := make(map[string]string)
	if c.Metadata != nil {
		c.Metadata.Unmarshal(metadata)
	}
	return &drivertypes.ClusterInfo{
		ClientCertificate:   c.ClientCertificate,
		ClientKey:           c.ClientKey,
		RootCaCertificate:   c.RootCaCertificate,
		ServiceAccountToken: c.ServiceAccountToken,
		Version:             c.K8sVersion,
		Endpoint:            c.ApiEndpoint,
		Config:              c.YkeConfig,
		Metadata:            metadata,
	}
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
		Metadata:            info.Metadata,
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
			return nil, fmt.Errorf("YKE node %q cluster id not equal '%s:%s'", nodeId, clusterId, c.Name, c.Id)
		}
		obj, err := NodeManager.FetchNodeById(nodeId)
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}
	return objs, nil
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
				c.SetStatus(nil, CLUSTER_STATUS_ERROR, err.Error())
				return false, nil
			}
			// The last f() call was a success, return cleanly
			return true, nil
		})
	}

	go run()
}

func (c *SCluster) addNodes(ctx context.Context, pendingNodes ...*SNode) error {
	return c.startAddNodes(ctx, pendingNodes...)
}

func (c *SCluster) startAddNodes(ctx context.Context, pendingNodes ...*SNode) error {
	ykeConf, err := ClusterManager.GetConfig(c, pendingNodes...)
	if err != nil {
		return err
	}
	if ykeConf == nil {
		return fmt.Errorf("YKE config is nil, maybe node already added???")
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

	createF := func() (err error) {
		err = c.SetStatus(nil, CLUSTER_STATUS_CREATING, "")
		if err != nil {
			return
		}

		info, err := driver.Create(ctx, opts, clusterInfo)
		if err != nil {
			log.Errorf("cluster create err: %v", err)
			return
		}

		if info != nil {
			err = c.saveClusterInfo(info)
			if err != nil {
				log.Errorf("Save cluster info after create error: %v", err)
				c.SetStatus(nil, CLUSTER_STATUS_ERROR, err.Error())
			}
		}
		setNodesStatus(pendingNodes, NODE_STATUS_RUNNING)
		return c.SetStatus(nil, CLUSTER_STATUS_RUNNING, "")
	}

	c.backoffTryRun(createF, 5)
	return nil
}

func (c *SCluster) Update(ctx context.Context, pendingNodes ...*SNode) error {
	ykeConf, err := ClusterManager.GetConfig(c, pendingNodes...)
	if err != nil {
		return err
	}
	if ykeConf == nil {
		log.Warningf("ykeConf is none, config not change, skip this update")
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

	updateF := func() (err error) {
		err = c.SetStatus(nil, CLUSTER_STATUS_UPDATING, "")
		if err != nil {
			return
		}

		info, err := driver.Update(ctx, opts, clusterInfo)
		if err != nil {
			log.Errorf("cluster update err: %v", err)
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
		return c.SetStatus(nil, CLUSTER_STATUS_RUNNING, "")
	}

	c.backoffTryRun(updateF, 5)
	return nil
}

func setNodesStatus(nodes []*SNode, status string) error {
	var err error
	for _, node := range nodes {
		err = node.SetStatus(nil, status, "")
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
		c.Metadata = jsonutils.Marshal(clusterInfo.Metadata)

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
	confObj, err := utils.ConvertToYkeConfig(confStr)
	if err != nil {
		return nil, err
	}
	return &confObj, err
}

func (c *SCluster) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return c.IsOwner(userCred)
}

func (c *SCluster) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("Cluster delete do nothing")
	return nil
}

func (c *SCluster) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return c.SVirtualResourceBase.Delete(ctx, userCred)
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
	return c.startRemoveCluster(ctx, userCred)
}

func (c *SCluster) startRemoveCluster(ctx context.Context, userCred mcclient.TokenCredential) error {
	deleteF := func() (err error) {
		log.Infof("Deleting cluster [%s]", c.Name)
		c.SetStatus(nil, CLUSTER_STATUS_DELETING, "")
		err = c.ClusterDriver().Remove(ctx, c.ToInfo())
		if err != nil {
			log.Errorf("Delete cluster error: %v", err)
			return
		}
		return
	}

	c.backoffTryRun(deleteF, 5)
	return nil
}

func (c *SCluster) ClusterDriver() drivertypes.Driver {
	driver := clusterdriver.Drivers["yke"]
	return driver
}

func (c *SCluster) GetYKEAuthzConfig() yketypes.AuthzConfig {
	return yketypes.AuthzConfig{Mode: ykecluster.DefaultAuthorizationMode}
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
	}
	return conf
}

func (c *SCluster) GetK8sWebhookAuthUrl() string {
	// TODO: impl this
	return "https://10.168.222.183:8443/webhook"
}

func (c *SCluster) GetYKEWebhookAuthConfig() yketypes.WebhookAuth {
	return yketypes.WebhookAuth{
		URL: c.GetK8sWebhookAuthUrl(),
	}
}

func (c *SCluster) GetClusterCIDR() string {
	cidr := c.ClusterCidr
	if len(cidr) == 0 {
		return DEFAULT_CLUSER_CIDR
	}
	return cidr
}

func (c *SCluster) GetServiceClusterIPRange() string {
	// TODO: different from c.ClusterCidr
	return c.GetClusterCIDR()
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
			},
		},
		PodSecurityPolicy:     false,
		ServiceClusterIPRange: c.GetServiceClusterIPRange(),
	}

	config.KubeController = yketypes.KubeControllerService{
		BaseService:           yketypes.BaseService{Image: images.Kubernetes},
		ServiceClusterIPRange: c.GetServiceClusterIPRange(),
		ClusterCIDR:           c.GetClusterCIDR(),
	}

	config.Scheduler = yketypes.SchedulerService{
		BaseService: yketypes.BaseService{Image: images.Kubernetes},
	}

	infraContainerImage := c.InfraContainerImage
	if len(infraContainerImage) == 0 {
		infraContainerImage = images.PodInfraContainer
	}

	config.Kubelet = yketypes.KubeletService{
		BaseService:         yketypes.BaseService{Image: images.Kubernetes},
		ClusterDomain:       c.GetClusterDomain(),
		ClusterDNSServer:    c.GetClusterDNSServiceIp(),
		InfraContainerImage: infraContainerImage,
	}

	config.Kubeproxy = yketypes.KubeproxyService{
		BaseService: yketypes.BaseService{Image: images.Kubernetes},
	}

	return config, nil
}

func (c *SCluster) NewYKEConfig() (yketypes.KubernetesEngineConfig, error) {
	conf := yketypes.KubernetesEngineConfig{}
	systemImages, err := c.GetYKESystemImages()
	if err != nil {
		return conf, err
	}
	conf.SystemImages = *systemImages
	conf.Authorization = c.GetYKEAuthzConfig()
	conf.Network = c.GetYKENetworkConfig()
	conf.Services, err = c.GetYKEServicesConfig(conf.SystemImages)
	if err != nil {
		return conf, err
	}
	conf.WebhookAuth = c.GetYKEWebhookAuthConfig()
	return conf, nil
}

func (c *SCluster) AllowPerformGenerateKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return c.IsOwner(userCred)
}

func (c *SCluster) PerformGenerateKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// TODO:
	// 1. Get normal user config
	// 2. Support Endpoint transparent proxy
	conf, err := c.GetAdminKubeconfig()
	if err != nil {
		return nil, httperrors.NewInternalServerError("Generate kubeconfig err: %v", err)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(conf), "kubeconfig")
	return ret, nil
}

func (c *SCluster) GetAdminKubeconfig() (string, error) {
	info, err := DecodeClusterInfo(c.ToInfo())
	if err != nil {
		return "", err
	}
	config := pki.GetKubeConfigX509WithData(info.Endpoint, c.Name, "kube-admin", info.RootCaCertificate, info.ClientCertificate, info.ClientKey)
	return config, nil
}

func (c *SCluster) AllowGetDetailsEngineConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.IsOwner(userCred)
}

func (c *SCluster) GetDetailsEngineConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	configStr := c.YkeConfig
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(configStr), "config")
	return ret, nil
}
