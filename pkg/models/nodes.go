package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	cloudmodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"
	ykehosts "yunion.io/x/yke/pkg/hosts"
	yketypes "yunion.io/x/yke/pkg/types"

	drivertypes "yunion.io/x/yunion-kube/pkg/clusterdriver/types"
	"yunion.io/x/yunion-kube/pkg/types/apis"
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
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(SNode{}, "nodes_tbl", "kube_node", "kube_nodes"),
	}
}

type SNodeManager struct {
	db.SStatusStandaloneResourceBaseManager
	cloudmodels.SInfrastructureManager
}

type SNode struct {
	db.SStatusStandaloneResourceBase
	cloudmodels.SInfrastructure

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

func validateHost(ctx context.Context, m *SNodeManager, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (string, error) {
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

	cloudHost := apis.CloudHost{}
	err = ret.Unmarshal(&cloudHost)
	if err != nil {
		return "", err
	}

	err = validateCloudHostCondition(cloudHost)
	if err != nil {
		return "", err
	}

	if name == "" {
		name = cloudHost.Name
		data.Set("name", jsonutils.NewString(name))
	}
	n, _ := NodeManager.FetchByIdOrName("", name)
	if n != nil {
		return "", httperrors.NewInputParameterError("Node name %q duplicate", name)
	}
	data.Set(CLOUD_HOST_DATA_KEY, jsonutils.Marshal(cloudHost))

	return cloudHost.Id, nil
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

func (m *SNodeManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := m.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	var sq *sqlchemy.SSubQuery
	if cluster, _ := query.GetString("cluster"); len(cluster) > 0 {
		clusters := ClusterManager.Query().SubQuery()
		sq = clusters.Query(clusters.Field("id")).
			Filter(sqlchemy.OR(
				sqlchemy.Equals(clusters.Field("name"), cluster),
				sqlchemy.Equals(clusters.Field("id"), cluster))).SubQuery()
	}
	if sq != nil {
		q = q.In("cluster_id", sq)
	}
	return q, nil
}

func NewNode(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*SNode, error) {
	data, err := NodeManager.ValidateCreateData(ctx, userCred, "", nil, data)
	if err != nil {
		return nil, err
	}
	model, err := db.NewModelObject(NodeManager)
	if err != nil {
		return nil, err
	}
	filterData := data.CopyIncludes(ModelCreateFields(NodeManager, userCred)...)
	err = filterData.Unmarshal(model)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	err = model.CustomizeCreate(ctx, userCred, "", nil, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return model.(*SNode), nil
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

	hostId, err := validateHost(ctx, m, userCred, data)
	if err != nil {
		return nil, err
	}
	data.Add(jsonutils.NewString(hostId), "host_id")

	return m.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
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

func (m *SNodeManager) FetchNodesByIds(ids []string) ([]*SNode, error) {
	ret := make([]*SNode, len(ids))
	for i, id := range ids {
		node, err := m.FetchNodeById(id)
		if err != nil {
			return nil, err
		}
		ret[i] = node
	}
	return ret, nil
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

func (n *SNode) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	dockerConf, _ := data.Get("dockerd_config")
	if dockerConf != nil {
		n.DockerdConfig = dockerConf
	}
	return n.SStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
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

func (n *SNode) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return allowPerformAction(ctx, userCred, query, data)
}

func (n *SNode) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := n.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	err = n.RemoveNodeFromCluster(ctx)
	if err != nil {
		return nil, err
	}
	return nil, n.RealDelete(ctx, userCred)
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
	return n.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (n *SNode) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Node delete do nothing")
	return nil
}

func (n *SNode) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return n.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (n *SNode) RemoveNodeFromCluster(ctx context.Context) error {
	cluster, err := n.GetCluster()
	if err != nil {
		return err
	}
	config, err := cluster.GetYKEConfig()
	if err != nil {
		return err
	}
	if config == nil {
		return nil
	}
	config = RemoveYKEConfigNode(config, n)
	return cluster.SetYKEConfig(config)
}

func (n *SNode) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return n.StartDeleteNodeTask(ctx, userCred, "", data)
}

func (n *SNode) StartDeleteNodeTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, data jsonutils.JSONObject) error {
	n.SetStatus(userCred, NODE_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "NodeDeleteTask", n, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (n *SNode) ToYKEHost() *ykehosts.Host {
	nodeConfig := n.GetNodeConfig()
	host := &ykehosts.Host{
		ConfigNode: *nodeConfig,
		IsEtcd:     n.Etcd,
		IsControl:  n.Controlplane,
		IsWorker:   n.Worker,
	}
	return host
}

func (n *SNode) GetDriver() (drivertypes.Driver, error) {
	cluster, err := n.GetCluster()
	if err != nil {
		return nil, err
	}
	return cluster.ClusterDriver(), nil
}

func (n *SNode) CleanUpComponents(ctx context.Context, data jsonutils.JSONObject) error {
	host := n.ToYKEHost()
	hostBytes, err := json.Marshal(host)
	if err != nil {
		return err
	}
	opts := &drivertypes.DriverOptions{
		StringOptions: map[string]string{"host": string(hostBytes)},
	}

	driver, err := n.GetDriver()
	if err != nil {
		return err
	}
	return driver.RemoveNode(ctx, opts)
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
			LiveRestore:     true,
			Graph:           DEFAULT_DOCKER_GRAPH_DIR,
			RegistryMirrors: DEFAULT_DOCKER_REGISTRY_MIRRORS,
		}, nil
	}
	config := apis.DockerdConfig{}
	err := n.DockerdConfig.Unmarshal(&config)
	if err != nil {
		return apis.DockerdConfig{}, err
	}
	config.LiveRestore = true
	if config.Graph == "" {
		config.Graph = DEFAULT_DOCKER_GRAPH_DIR
	}
	if len(config.RegistryMirrors) == 0 {
		config.RegistryMirrors = DEFAULT_DOCKER_REGISTRY_MIRRORS
	}
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
	extra := n.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra.Add(jsonutils.NewString(n.getClusterName()), "cluster")
	extra.Add(jsonutils.NewString(rolesString(n)), "roles")
	return extra
}

func (n *SNode) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := n.SStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	extra.Add(jsonutils.NewString(n.getClusterName()), "cluster")
	extra.Add(jsonutils.NewString(rolesString(n)), "roles")
	return extra
}

func (n *SNode) StartAgentStartTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "NodeStartAgentTask", n, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (n *SNode) GetAgentRegisterConfig() (ret apis.HostRegisterConfig, error error) {
	serverUrl, err := GetKubeServerUrl()
	if err != nil {
		err = fmt.Errorf("Get server url: %v", err)
		return
	}
	dockerdConfig, err := n.GetDockerdConfig()
	if err != nil {
		err = fmt.Errorf("Get dockerd config: %v", err)
		return
	}
	ret = apis.HostRegisterConfig{
		AgentConfig: apis.AgentConfig{
			ServerUrl: serverUrl,
			Token:     n.ClusterId,
			ClusterId: n.ClusterId,
			NodeId:    n.Id,
		},
		DockerdConfig: dockerdConfig,
	}
	return
}

func (n *SNode) StartAgentOnHost(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	hostId := n.HostId
	if hostId == "" {
		log.Debugf("Not yunioncloud host, skip it")
		return
	}
	err := n.StartAgentStartTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
	if err != nil {
		log.Errorf("Start agent start task error: %v", err)
		n.SetStatus(userCred, NODE_STATUS_ERROR, err.Error())
	}
}

func (n *SNode) AllowGetDetailsDockerConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return n.AllowGetDetails(ctx, userCred, query)
}

func (n *SNode) GetDetailsDockerConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return n.DockerdConfig, nil
}

func (n *SNode) GetCloudHost() (*apis.CloudHost, error) {
	session, err := GetAdminSession()
	if err != nil {
		err = httperrors.NewInternalServerError("Get admin session: %v", err)
		return nil, err
	}
	ret, err := cloudmod.Hosts.Get(session, n.HostId, nil)
	if err != nil {
		return nil, err
	}

	cloudHost := apis.CloudHost{}
	err = ret.Unmarshal(&cloudHost)
	if err != nil {
		return nil, err
	}
	return &cloudHost, nil
}
