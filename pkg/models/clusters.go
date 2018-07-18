package models

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"yunion.io/yke/pkg/pki"
	"yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/httperrors"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/mcclient"
	"yunion.io/yunioncloud/pkg/sqlchemy"
	"yunion.io/yunioncloud/pkg/util/wait"

	"yunion.io/yunion-kube/pkg/clusterdriver"
	drivertypes "yunion.io/yunion-kube/pkg/clusterdriver/types"
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
	CLUSTER_PRE_CREATING = "pre-creating"
	CLUSTER_CREATING     = "creating"
	CLUSTER_POST_CHECK   = "post-checking"
	CLUSTER_RUNNING      = "running"
	CLUSTER_ERROR        = "error"
	CLUSTER_UPDATING     = "updating"
)

type SClusterManager struct {
	db.SVirtualResourceBaseManager
}

func (m *SClusterManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return m.SVirtualResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func (m *SClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	k8sVersion, _ := data.GetString("k8s_version")
	if k8sVersion == "" {
		k8sVersion = DEFAULT_K8S_VERSION
		data.Set("k8s_version", jsonutils.NewString(k8sVersion))
	}
	_, err := K8sYKEVersionMap.GetYKEVersion(k8sVersion)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Invalid version %q: %v", k8sVersion, err)
	}
	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (m *SClusterManager) FilterByOwner(q *sqlchemy.SQuery, ownerProjId string) *sqlchemy.SQuery {
	if len(ownerProjId) > 0 {
		q = q.Equals("tenant_id", ownerProjId)
	}
	return q
}

func (m *SClusterManager) FetchCluster(ident string) *SCluster {
	cluster, err := m.FetchByIdOrName("", ident)
	if err != nil {
		log.Errorf("Fetch cluster %q fail: %v", ident, err)
		return nil
	}
	return cluster.(*SCluster)
}

func (m *SClusterManager) GetCluster(ident string) (*apis.Cluster, error) {
	cluster := m.FetchCluster(ident)
	if cluster == nil {
		return nil, fmt.Errorf("Not found cluster %q", ident)
	}
	return cluster.Cluster()
}

func (m *SClusterManager) AddClusterNodes(clusterId string, pendingNodes ...*SNode) error {
	cluster := m.FetchCluster(clusterId)
	if cluster == nil {
		return ClusterNotFoundError
	}

	if slice.ContainsString([]string{CLUSTER_UPDATING, CLUSTER_POST_CHECK}, cluster.Status) {
		return fmt.Errorf("Cluster %q add node: status is %q", cluster.Name, cluster.Status)
	}

	return cluster.AddNodes(context.Background(), pendingNodes...)
}

func (m *SClusterManager) UpdateCluster(clusterId string, pendingNodes ...*SNode) error {
	cluster := m.FetchCluster(clusterId)
	if cluster == nil {
		return ClusterNotFoundError
	}
	if slice.ContainsString([]string{CLUSTER_UPDATING, CLUSTER_POST_CHECK}, cluster.Status) {
		return fmt.Errorf("Cluster %q update: status is %s", cluster.Name, cluster.Status)
	}

	return cluster.Update(context.Background(), pendingNodes...)
}

func (m *SClusterManager) getSpec(cluster *SCluster, pendingNodes ...*SNode) (*types.KubernetesEngineConfig, error) {
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

func (m *SClusterManager) getConfig(reconcileYKE bool, cluster *SCluster, pendingNodes ...*SNode) (old, new *types.KubernetesEngineConfig, err error) {
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
		log.Errorf("======nodes: %#v, len(%d)", nodes, len(nodes))
		systemImages, err := cluster.GetYKESystemImages()
		if err != nil {
			return nil, nil, err
		}
		newConf := types.KubernetesEngineConfig{}
		if old != nil {
			newConf = *old
		}
		newConf.Nodes = nodes
		newConf.SystemImages = *systemImages
		new = &newConf
		log.Infof("======get newconfig: %#v, images: %#v", new, systemImages)
	}
	return old, new, nil
}

func (m *SClusterManager) reconcileYKENodes(clusterId string, pendingNodes ...*SNode) ([]types.ConfigNode, error) {
	objs, err := NodeManager.ListByCluster(clusterId)
	if err != nil {
		return nil, err
	}
	log.Errorf("******* before merge objs: %#v, pendingNodes: %#v", objs, pendingNodes)
	objs = mergePendingNodes(objs, pendingNodes)
	log.Errorf("******* after merge objs: %#v, pendingNodes: %#v", objs, pendingNodes)
	etcd := false
	controlplane := false
	var nodes []types.ConfigNode
	for _, obj := range objs {
		machine, err := obj.Node()
		if err != nil {
			return nil, err
		}
		log.Infof("===check node config %#v", machine.NodeConfig)
		if slice.ContainsString(machine.NodeConfig.Role, "etcd") {
			etcd = true
		}
		if slice.ContainsString(machine.NodeConfig.Role, "controlplane") {
			controlplane = true
		}

		node := *machine.NodeConfig
		if node.User == "" {
			node.User = "root"
		}
		if node.Port == "" {
			node.Port = "22"
		}
		if node.NodeName == "" {
			node.NodeName = machine.Name
		}
		nodes = append(nodes, node)
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
	Mode                string               `nullable:"false" create:"required" list:"user"`
	K8sVersion          string               `nullable:"true" create:"required" list:"user"`
	ApiEndpoint         string               `nullable:"true" list:"user"`
	ClientCertificate   string               `nullable:"true"`
	ClientKey           string               `nullable:"true"`
	RootCaCertificate   string               `nullable:"true"`
	ServiceAccountToken string               `nullable:"true"`
	Certs               string               `nullable:"true"`
	YkeConfig           string               `nullable:"true"`
	Metadata            jsonutils.JSONObject `nullable:"true"`
}

func (c *SCluster) GetYKESystemImages() (*types.SystemImages, error) {
	if c.K8sVersion == "" {
		c.K8sVersion = DEFAULT_K8S_VERSION
	}
	ykeVersion, err := K8sYKEVersionMap.GetYKEVersion(c.K8sVersion)
	if err != nil {
		return nil, err
	}
	imageDefaults := types.K8sVersionToSystemImages[ykeVersion]
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
		//Status: c.Status,
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

func (c *SCluster) AddNodes(ctx context.Context, pendingNodes ...*SNode) error {
	if c.Status == CLUSTER_POST_CHECK {
		return nil
	}
	return c.startAddNodes(ctx, pendingNodes...)
}

func (c *SCluster) startAddNodes(ctx context.Context, pendingNodes ...*SNode) error {
	ykeConf, err := ClusterManager.getSpec(c, pendingNodes...)
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

	c.SetStatus(nil, CLUSTER_CREATING, "")

	createF := func() (done bool, err error) {
		info, err := driver.Create(ctx, opts, clusterInfo)
		if err != nil {
			log.Errorf("cluster driver create err: %v", err)
			done = false
			c.SetStatus(nil, CLUSTER_ERROR, "")
			return
		}

		if info != nil {
			err = c.saveClusterInfo(info)
			if err != nil {
				log.Errorf("Save cluster info after create error: %v", err)
				c.SetStatus(nil, CLUSTER_ERROR, "")
			}
		}
		c.SetStatus(nil, CLUSTER_RUNNING, "")
		done = true
		//err = nil
		return
	}

	go func() {
		timeOut := 30 * time.Minute
		interval := 5 * time.Second
		err := wait.Poll(interval, timeOut, createF)
		if err != nil {
			log.Errorf("Create poll error: %v", err)
		}
	}()
	return nil
}

func (c *SCluster) Update(ctx context.Context, pendingNodes ...*SNode) error {
	ykeConf, err := ClusterManager.getSpec(c, pendingNodes...)
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

	c.SetStatus(nil, CLUSTER_UPDATING, "")

	updateF := func() (done bool, err error) {
		log.Debugf("=====start update====")
		info, err := driver.Update(ctx, opts, clusterInfo)
		if err != nil {
			log.Errorf("cluster driver update err: %v", err)
			done = false
			c.SetStatus(nil, CLUSTER_ERROR, "")
			return
		}

		if info != nil {
			err = c.saveClusterInfo(info)
			if err != nil {
				log.Errorf("Save cluster info after update error: %v", err)
				c.SetStatus(nil, CLUSTER_ERROR, "")
				return
			}
		}
		c.SetStatus(nil, CLUSTER_RUNNING, "")
		done = true
		return
	}

	go func() {
		timeOut := 30 * time.Minute
		interval := 5 * time.Second
		err := wait.Poll(interval, timeOut, updateF)
		if err != nil {
			log.Errorf("Update poll error: %v", err)
		}
	}()
	return nil
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

func (c *SCluster) SetYKEConfig(config *types.KubernetesEngineConfig) error {
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

func (c *SCluster) GetYKEConfig() (conf *types.KubernetesEngineConfig, err error) {
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
	deleteF := func() (done bool, err error) {
		log.Infof("Deleting cluster [%s]", c.Name)
		time.Sleep(5 * time.Second)
		for i := 0; i < 4; i++ {
			err = c.ClusterDriver().Remove(ctx, c.ToInfo())
			if err == nil {
				break
			}
			if i == 3 {
				log.Errorf("failed to remove the cluster [%s]: %v", c.Name, err)
				//return
				break
			}
			time.Sleep(1 * time.Second)
		}
		log.Infof("Deleted cluster [%s]", c.Name)
		if err != nil {
			log.Errorf("Delete cluster error: %v", err)
			c.SetStatus(nil, CLUSTER_ERROR, "")
			//err = nil
		} else {
			done = true
		}
		return
	}

	go func() {
		timeOut := 30 * time.Minute
		interval := 5 * time.Second
		err := wait.Poll(interval, timeOut, deleteF)
		if err != nil {
			log.Errorf("Delete poll error: %v", err)
		}
	}()
	return nil
}

func (c *SCluster) ClusterDriver() drivertypes.Driver {
	driver := clusterdriver.Drivers["yke"]
	return driver
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
