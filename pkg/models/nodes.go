package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	yketypes "yunion.io/yke/pkg/types"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	cloudmodels "yunion.io/yunioncloud/pkg/compute/models"
	"yunion.io/yunioncloud/pkg/httperrors"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/mcclient"
	cloudmod "yunion.io/yunioncloud/pkg/mcclient/modules"
	"yunion.io/yunioncloud/pkg/sqlchemy"
	"yunion.io/yunioncloud/pkg/util/sets"

	"yunion.io/yunion-kube/pkg/request"
	"yunion.io/yunion-kube/pkg/types/apis"
	"yunion.io/yunion-kube/pkg/types/slice"
)

var NodeManager *SNodeManager

var (
	NodeNotFoundError = errors.New("Node not found")
)

const (
	NODE_STATUS_INIT     = "init"
	NODE_STATUS_READY    = "ready"
	NODE_STATUS_DEPLOY   = "deploying"
	NODE_STATUS_RUNNING  = "running"
	NODE_STATUS_ERROR    = "error"
	NODE_STATUS_UPDATING = "updating"
	NODE_STATUS_DELETING = "deleting"
)

func init() {
	NodeManager = &SNodeManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SNode{}, "nodes_tbl", "node", "nodes"),
	}
}

type SNodeManager struct {
	db.SVirtualResourceBaseManager
}

func (m *SNodeManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return m.SVirtualResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func validateRoles(data jsonutils.JSONObject) (etcd, ctrl, worker bool, err error) {
	validRoles := sets.NewString("etcd", "controlplane", "worker")
	roles, err := data.GetArray("roles")
	if err != nil {
		return
	}
	if len(roles) == 0 {
		err = fmt.Errorf("Roles must provided")
		return
	}
	var role string
	for _, reqRole := range roles {
		role, err = reqRole.GetString()
		if err != nil {
			return
		}
		if !validRoles.Has(role) {
			err = fmt.Errorf("Invalid role %s", role)
			return
		}
		switch role {
		case "etcd":
			etcd = true
		case "controlplane":
			ctrl = true
		case "worker":
			worker = true
		}
	}
	if !(etcd || ctrl || worker) {
		err = fmt.Errorf("Invalid roles: %s", roles)
	}
	return
}

func validateHost(m *SNodeManager, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (string, error) {
	name, _ := data.GetString("name")
	hostId, _ := data.GetString("host")
	if name == "" && hostId == "" {
		return "", httperrors.NewInputParameterError("One of host or name must specified")
	}
	if hostId == "" {
		return "", nil
	}
	session, err := GetAdminSession()
	if err != nil {
		return "", httperrors.NewInternalServerError("Get admin session: %v", err)
	}
	ret, err := cloudmod.Hosts.Get(session, hostId, nil)
	if err != nil {
		return "", err
	}
	id, _ := ret.GetString("id")
	if id == "" {
		return "", httperrors.NewNotFoundError("Host %q not found", hostId)
	}
	hostType, _ := ret.GetString("host_type")
	if !slice.ContainsString([]string{cloudmodels.HOST_TYPE_HYPERVISOR, cloudmodels.HOST_TYPE_KUBELET}, hostType) {
		return "", httperrors.NewInputParameterError("Host %q type %q not support", hostId, hostType)
	}
	hostName, _ := ret.GetString("name")

	node, err := m.FetchNodeByHostId(id)
	if err != nil && err != sql.ErrNoRows {
		return "", httperrors.NewInternalServerError("Fetch node by host id %q: %v", id, err)
	}
	if node != nil {
		return "", httperrors.NewInputParameterError("Host %q already used by node %q", hostId, node.Name)
	}

	if name == "" {
		data.Set("name", jsonutils.NewString(hostName))
	}

	return id, nil
}

func validateDockerConfig(data *jsonutils.JSONDict) error {
	obj, _ := data.Get("dockerd_config")
	if obj == nil {
		return nil
	}

	config := apis.DockerdConfig{}
	err := obj.Unmarshal(&config)
	if err != nil {
		return httperrors.NewInputParameterError("Parse registryMirrors %s error: %v", obj, err)
	}
	return nil
}

func (m *SNodeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	clusterIdent, _ := data.GetString("cluster")
	if clusterIdent == "" {
		return nil, httperrors.NewInputParameterError("Cluster must specified")
	}
	cluster, err := ClusterManager.FetchClusterByIdOrName(ownerId, clusterIdent)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Cluster %q found error: %v", clusterIdent, err)
	}
	data.Add(jsonutils.NewString(cluster.Id), "cluster_id")

	err = validateDockerConfig(data)
	if err != nil {
		return nil, err
	}

	isEtcd, isCtrl, isWorker, err := validateRoles(data)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Cluster role: %v", err)
	}
	toBool := func(v bool) jsonutils.JSONObject {
		if v {
			return jsonutils.JSONTrue
		}
		return jsonutils.JSONFalse
	}
	data.Add(toBool(isEtcd), "etcd")
	data.Add(toBool(isCtrl), "controlplane")
	data.Add(toBool(isWorker), "worker")

	hostId, err := validateHost(m, userCred, data)
	if err != nil {
		return nil, err
	}
	data.Add(jsonutils.NewString(hostId), "host_id")

	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (m *SNodeManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	nodes := make([]*SNode, len(items))
	for i, t := range items {
		nodes[i] = t.(*SNode)
	}
	for _, n := range nodes {
		TaskManager().Run(func() {
			n.StartAgentOnHost(ctx, userCred, query, data)
		}, nil)
	}
}

func (m *SNodeManager) FetchNodeById(ident string) (*SNode, error) {
	node, err := m.FetchById(ident)
	if err != nil {
		log.Errorf("Fetch node %q fail: %v", ident, err)
		if err == sql.ErrNoRows {
			return nil, NodeNotFoundError
		}
		return nil, err
	}
	return node.(*SNode), nil
}

func (m *SNodeManager) FetchNodeByHostId(hostId string) (*SNode, error) {
	if hostId == "" {
		return nil, fmt.Errorf("Host id not provided")
	}
	nodes := m.Query().SubQuery()
	q := nodes.Query().Filter(sqlchemy.Equals(nodes.Field("host_id"), hostId))
	node := SNode{}
	err := q.First(&node)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (m *SNodeManager) FetchClusterNode(clusterId, ident string) (*SNode, error) {
	nodes, err := m.ListByCluster(clusterId)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		if node.Name == ident || node.Id == ident {
			return node, nil
		}
	}
	log.Errorf("Cluster %q Node %q not found", clusterId, ident)
	return nil, NodeNotFoundError
}

func (m *SNodeManager) GetNodeById(cluster, ident string) (*apis.Node, error) {
	node, err := m.FetchNodeById(ident)
	if err != nil {
		return nil, err
	}
	return node.Node()
}

func (m *SNodeManager) ListByCluster(clusterId string) ([]*SNode, error) {
	nodes := NodeManager.Query().SubQuery()
	q := nodes.Query().Filter(sqlchemy.Equals(nodes.Field("cluster_id"), clusterId))
	objs := make([]SNode, 0)
	err := db.FetchModelObjects(NodeManager, q, &objs)
	if err != nil {
		return nil, err
	}
	return ConvertPtrNodes(objs), nil
}

func ConvertPtrNodes(objs []SNode) []*SNode {
	ret := make([]*SNode, len(objs))
	for i, obj := range objs {
		temp := obj
		ret[i] = &temp
	}
	return ret
}

func mergePendingNodes(nodes, pendingNodes []*SNode) []*SNode {
	isIn := func(pnode *SNode, nodes []*SNode) (int, bool) {
		for idx, node := range nodes {
			if node.Id == pnode.Id {
				return idx, true
			}
		}
		return 0, false
	}

	for _, pnode := range pendingNodes {
		if idx, ok := isIn(pnode, nodes); ok {
			nodes[idx] = pnode
		} else {
			nodes = append(nodes, pnode)
		}
	}

	return nodes
}

type SNode struct {
	db.SVirtualResourceBase

	ClusterId        string `nullable:"false" create:"required" list:"user"`
	Etcd             bool   `nullable:"true" create:"required" list:"user"`
	Controlplane     bool   `nullable:"true" create:"required" list:"user"`
	Worker           bool   `nullable:"true" create:"required" list:"user"`
	HostnameOverride string `nullable:"true" create:"optional" list:"user"`
	HostId           string `nullable:"true" create:"optional" list:"user"`

	Address           string `nullable:"true" list:"user"`
	RequestedHostname string `nullable:"true" list:"user"`

	DockerdConfig jsonutils.JSONObject `nullable:"true" list:"user"`
	Labels        jsonutils.JSONObject `nullable:"true" list:"user"`
	DockerInfo    jsonutils.JSONObject `nullable:"true" list:"user"`
}

func (n *SNode) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	dockerConf, _ := data.Get("dockerd_config")
	if dockerConf != nil {
		n.DockerdConfig = dockerConf
	}
	return n.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

// Register set node status to ready, means node is ready for deploy
func (n *SNode) Register(data *apis.Node) (*SNode, error) {
	if n.ClusterId != data.ClusterId {
		return nil, fmt.Errorf("ClusterId %q and %q not match", n.ClusterId, data.ClusterId)
	}

	if data.Address == "" {
		return nil, fmt.Errorf("Address must provided")
	}

	_, err := n.GetModelManager().TableSpec().Update(n, func() error {
		n.Address = data.Address
		n.RequestedHostname = data.RequestedHostname
		if data.DockerInfo != nil {
			dInfo := jsonutils.Marshal(data.DockerInfo)
			n.DockerInfo = dInfo
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if n.Status != NODE_STATUS_RUNNING {
		n.SetStatus(GetAdminCred(), NODE_STATUS_READY, "")
	}

	return n, nil
}

func (n *SNode) Node() (*apis.Node, error) {
	return &apis.Node{
		Name:         n.Name,
		Etcd:         n.Etcd,
		ControlPlane: n.Controlplane,
		Worker:       n.Worker,
		NodeConfig:   n.GetNodeConfig(),
	}, nil
}

func (n *SNode) GetRoles() []string {
	roles := []string{}
	if n.Etcd {
		roles = append(roles, "etcd")
	}
	if n.Controlplane {
		roles = append(roles, "controlplane")
	}
	if n.Worker {
		roles = append(roles, "worker")
	}
	return roles
}

func (n *SNode) GetLabels() (map[string]string, error) {
	labels := make(map[string]string)
	var err error
	if n.Labels != nil {
		err = n.Labels.Unmarshal(labels)
	}
	return labels, err
}

func (n *SNode) YKENodeName() string {
	return fmt.Sprintf("%s:%s", n.ClusterId, n.Id)
}

func (n *SNode) GetNodeConfig() *yketypes.ConfigNode {
	hostnameOverride := n.HostnameOverride
	if len(hostnameOverride) == 0 {
		hostnameOverride = n.RequestedHostname
	}
	node := &yketypes.ConfigNode{
		NodeName:         n.YKENodeName(),
		HostnameOverride: hostnameOverride,
		Address:          n.Address,
		Port:             "22",
		User:             "root",
		Role:             n.GetRoles(),
		DockerSocket:     "/var/run/docker.sock",
	}
	labels, err := n.GetLabels()
	if err != nil {
		log.Errorf("Get labels error: %v", err)
	} else {
		node.Labels = labels
	}
	return node
}

func (n *SNode) GetCluster() (*SCluster, error) {
	return ClusterManager.FetchClusterById(n.ClusterId)
}

func (n *SNode) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return n.IsOwner(userCred)
}

func (n *SNode) ValidateDeleteCondition(ctx context.Context) error {
	//cluster, err := n.GetCluster()
	//if err != nil {
	//return err
	//}
	//if sets.NewString(CLUSTER_CREATING, CLUSTER_POST_CHECK, CLUSTER_UPDATING).Has(cluster.Status) {
	//return fmt.Errorf("Can't delete node when cluster %q status is %q", cluster.Name, cluster.Status)
	//}
	//return nil
	return n.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (n *SNode) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	cluster, err := n.GetCluster()
	if err != nil {
		return err
	}
	config, _, err := ClusterManager.getConfig(cluster)
	if err != nil {
		return err
	}
	if config == nil {
		return nil
	}
	config = RemoveYKEConfigNode(config, n)
	return cluster.SetYKEConfig(config)
}

func RemoveYKEConfigNode(config *yketypes.KubernetesEngineConfig, rNode *SNode) *yketypes.KubernetesEngineConfig {
	ykeNodes := config.Nodes
	if len(ykeNodes) == 0 {
		return config
	}
	newNodes := make([]yketypes.ConfigNode, 0)
	for _, n := range ykeNodes {
		if n.NodeName == rNode.YKENodeName() {
			continue
		}
		newNodes = append(newNodes, n)
	}
	config.Nodes = newNodes
	return config
}

func (n *SNode) GetDockerdConfig() (apis.DockerdConfig, error) {
	if n.DockerdConfig == nil {
		return apis.DockerdConfig{
			// Enable LiveRestore by default
			LiveRestore: true,
			Graph:       "/opt/docker",
		}, nil
	}
	config := apis.DockerdConfig{}
	err := n.DockerdConfig.Unmarshal(&config)
	if err != nil {
		return apis.DockerdConfig{}, err
	}
	config.LiveRestore = true
	return config, err
}

func (n *SNode) getClusterName() string {
	cluster, _ := n.GetCluster()
	if cluster == nil {
		return ""
	}
	return cluster.Name
}

func rolesString(n *SNode) string {
	return strings.Join(n.GetRoles(), ",")
}

func (n *SNode) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := n.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra.Add(jsonutils.NewString(n.getClusterName()), "cluster")
	extra.Add(jsonutils.NewString(rolesString(n)), "roles")
	return extra
}

func (n *SNode) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := n.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	extra.Add(jsonutils.NewString(n.getClusterName()), "cluster")
	extra.Add(jsonutils.NewString(rolesString(n)), "roles")
	return extra
}

func validateHostInfo(host apis.CloudHost) (err error) {
	log.Infof("Get hosts: %#v", host)
	if host.ManagerUrl == "" {
		err = fmt.Errorf("Host %q not found manager_uri", host.Id)
		return
	}

	if !host.Enabled {
		err = fmt.Errorf("Host %q not enable", host.Id)
		return
	}

	if host.Status != "running" {
		err = fmt.Errorf("Host %q status %q is not 'running'", host.Id, host.Status)
		return
	}

	if host.HostStatus != "online" {
		err = fmt.Errorf("Host %q host_status %q is not 'online'", host.Id, host.HostStatus)
		return
	}
	return nil
}

func (n *SNode) StartAgentOnHost(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	hostId := n.HostId
	if hostId == "" {
		log.Debugf("Not yunioncloud host, skip it")
		return
	}
	session, err := GetAdminSession()
	if err != nil {
		n.SetStatus(userCred, NODE_STATUS_ERROR, fmt.Sprintf("Get admin session: %v", err))
		return
	}
	result, err := cloudmod.Hosts.Get(session, hostId, nil)
	if err != nil {
		n.SetStatus(userCred, NODE_STATUS_ERROR, fmt.Sprintf("Get host id %q: %v", hostId, err))
		return
	}
	cloudHost := apis.CloudHost{}
	err = result.Unmarshal(&cloudHost)
	if err != nil {
		n.SetStatus(userCred, NODE_STATUS_ERROR, fmt.Sprintf("Unmarshal error: %v", err))
		return
	}
	err = validateHostInfo(cloudHost)
	if err != nil {
		n.SetStatus(userCred, NODE_STATUS_ERROR, fmt.Sprintf("HostInfo error: %v", err))
		return
	}
	url := "/kubeagent/start"
	serverUrl, err := GetKubeServerUrl()
	if err != nil {
		n.SetStatus(userCred, NODE_STATUS_ERROR, fmt.Sprintf("Get server url: %v", err))
		return
	}
	dockerdConfig, err := n.GetDockerdConfig()
	if err != nil {
		n.SetStatus(userCred, NODE_STATUS_ERROR, fmt.Sprintf("Get dockerd config: %v", err))
		return
	}
	registerConfig := apis.HostRegisterConfig{
		AgentConfig: apis.AgentConfig{
			ServerUrl: serverUrl,
			Token:     n.ClusterId,
			ClusterId: n.ClusterId,
			NodeId:    n.Id,
		},
		DockerdConfig: dockerdConfig,
	}
	body := jsonutils.Marshal(registerConfig)
	_, err = request.Post(cloudHost.ManagerUrl, userCred.GetTokenString(), url, nil, body)
	if err != nil {
		n.SetStatus(userCred, NODE_STATUS_ERROR, fmt.Sprintf("Start agent on host %q: %v", hostId, err))
	}
}

func (n *SNode) AllowGetDetailsDockerConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return n.IsOwner(userCred)
}

func (n *SNode) GetDetailsDockerConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return n.DockerdConfig, nil
}
