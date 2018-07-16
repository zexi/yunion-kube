package models

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/mcclient"

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
	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
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

func (m *SClusterManager) UpdateCluster(node *SNode, isCreate bool) (*SCluster, error) {
	cluster := m.FetchCluster(node.ClusterId)
	if cluster == nil {
		return nil, ClusterNotFoundError
	}
	//spec, err := m.getSpec(cluster)
	//if err != nil || spec == nil {
	//return cluster, err
	//}
	if isCreate {
		err := m.createInner(cluster)
		if err != nil {
			return nil, err
		}
	}
	return cluster, nil
}

func (m *SClusterManager) createInner(cls *SCluster) error {
	if slice.ContainsString([]string{CLUSTER_UPDATING, CLUSTER_RUNNING, CLUSTER_POST_CHECK}, cls.Status) {
		err := fmt.Errorf("Cluster %s status is %s", cls.Name, cls.Status)
		log.Infof("%v", err)
		return err
	}
	return cls.create(context.Background())
}

func (m *SClusterManager) getSpec(cluster *SCluster) (*types.KubernetesEngineConfig, error) {
	oldConf, _, err := m.getConfig(false, cluster)
	if err != nil {
		return nil, err
	}
	_, newConf, err := m.getConfig(true, cluster)
	if err != nil {
		return nil, err
	}
	log.Warningf("==oldConf: %#v, newConf: %#v", oldConf, newConf)
	if reflect.DeepEqual(oldConf, newConf) {
		newConf = nil
	}
	return newConf, nil
}

func (m *SClusterManager) getConfig(reconcileYKE bool, cluster *SCluster) (old, new *types.KubernetesEngineConfig, err error) {
	clusterId := cluster.Id
	old, err = cluster.GetYunionKubernetesEngineConfig()
	if err != nil {
		return nil, nil, err
	}
	if reconcileYKE {
		nodes, err := m.reconcileYKENodes(clusterId)
		if err != nil {
			return nil, nil, err
		}
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

func (m *SClusterManager) reconcileYKENodes(clusterId string) ([]types.ConfigNode, error) {
	machines, err := NodeManager.ListByCluster(clusterId)
	if err != nil {
		return nil, err
	}
	etcd := false
	controlplane := false
	var nodes []types.ConfigNode
	for _, machine := range machines {
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
	Mode                         string               `nullable:"false" create:"required" list:"admin"`
	ApiEndpoint                  string               `nullable:"true" list:"user"`
	ClientCertificate            string               `nullable:"true"`
	ClientKey                    string               `nullable:"true"`
	RootCaCertificate            string               `nullable:"true"`
	ServiceAccountToken          string               `nullable:"true"`
	Certs                        string               `nullable:"true"`
	Version                      string               `nullable:"true"`
	YunionKubernetesEngineConfig string               `nullable:"true", list: "admin"`
	Metadata                     jsonutils.JSONObject `nullable:"true", list: "admin"`

	Spec          jsonutils.JSONObject `nullable:"true" list:"admin"`
	ClusterStatus jsonutils.JSONObject `nullable:"true" list:"admin"`
}

func (c *SCluster) GetYKESystemImages() (*types.SystemImages, error) {
	imageDefaults := types.K8sVersionToSystemImages[types.K8sV110]
	return &imageDefaults, nil
}

func (c *SCluster) Cluster() (*apis.Cluster, error) {
	spec := apis.ClusterSpec{}
	status := apis.ClusterStatus{}
	if c.Spec != nil {
		c.Spec.Unmarshal(&spec)
	}
	if c.ClusterStatus != nil {
		c.ClusterStatus.Unmarshal(&status)
	}

	ykeConf, err := c.GetYunionKubernetesEngineConfig()
	if err != nil {
		return nil, err
	}

	return &apis.Cluster{
		Id:                           c.Id,
		Name:                         c.Name,
		Spec:                         spec,
		Status:                       status,
		CaCert:                       c.RootCaCertificate,
		ApiEndpoint:                  c.ApiEndpoint,
		YunionKubernetesEngineConfig: ykeConf,
	}, nil
}

func (c *SCluster) ToInfo() *drivertypes.ClusterInfo {
	//metadata := make(map[string]string)
	//c.Metadata.Unmarshal(metadata)
	return &drivertypes.ClusterInfo{
		ClientCertificate:   c.ClientCertificate,
		ClientKey:           c.ClientKey,
		RootCaCertificate:   c.RootCaCertificate,
		ServiceAccountToken: c.ServiceAccountToken,
		Version:             c.Version,
		Endpoint:            c.ApiEndpoint,
		//Metadata:            metadata,
		Status: c.Status,
	}
}

func (c *SCluster) create(ctx context.Context) error {
	if c.Status == CLUSTER_POST_CHECK {
		return nil
	}

	// update status to creating
	ykeConf, err := ClusterManager.getSpec(c)
	if err != nil {
		return err
	}
	if ykeConf == nil {
		return fmt.Errorf("YKE config is nil")
	}
	ykeConfStr, err := utils.ConvertYkeConfigToStr(*ykeConf)
	if err != nil {
		return err
	}

	opts := &drivertypes.DriverOptions{
		StringOptions: map[string]string{"ykeConfig": ykeConfStr},
	}

	driver := clusterdriver.Drivers["yke"]
	clusterInfo := c.ToInfo()

	go func() {
		time.Sleep(5 * time.Second)
		log.Debugf("=====start create====")
		info, err := driver.Create(ctx, opts, clusterInfo)
		if err != nil {
			log.Errorf("cluster driver create err: %v", err)
			return
		}

		if info != nil {
			c.transformClusterInfo(info)
		}
	}()

	return nil
}

func (c *SCluster) transformClusterInfo(clusterInfo *drivertypes.ClusterInfo) {
	log.Infof("=====transformClusterInfo: %#v", clusterInfo)
	c.ClientCertificate = clusterInfo.ClientCertificate
	c.ClientKey = clusterInfo.ClientKey
	c.RootCaCertificate = clusterInfo.RootCaCertificate
	c.Version = clusterInfo.Version
	c.ApiEndpoint = clusterInfo.Endpoint
}

func (c *SCluster) GetYunionKubernetesEngineConfig() (conf *types.KubernetesEngineConfig, err error) {
	confStr := c.YunionKubernetesEngineConfig
	if confStr == "" {
		return
	}
	confObj, err := utils.ConvertToYkeConfig(confStr)
	if err != nil {
		return nil, err
	}
	return &confObj, err
}
