package models

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	cloudmodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/utils"
	yketypes "yunion.io/x/yke/pkg/types"

	"yunion.io/x/yunion-kube/pkg/types/apis"
)

const (
	CLOUD_HOST_DATA_KEY   = "cloudHost"
	CLOUD_HOSTS_DATA_KEY  = "cloudHosts"
	NODES_CONFIG_DATA_KEY = "nodesConfig"
	NODES_DEPLOY_IDS_KEY  = "nodesId"
)

const (
	DEFAULT_DOCKER_GRAPH_DIR = "/opt/docker"

	DEFAULT_DOCKER_REGISTRY_MIRROR1 = "http://hub-mirror.c.163.com"
	DEFAULT_DOCKER_REGISTRY_MIRROR2 = "https://docker.mirrors.ustc.edu.cn"
	DEFAULT_DOCKER_REGISTRY_MIRROR3 = "https://registry.docker-cn.com"
)

var (
	DEFAULT_DOCKER_REGISTRY_MIRRORS = []string{DEFAULT_DOCKER_REGISTRY_MIRROR1, DEFAULT_DOCKER_REGISTRY_MIRROR2, DEFAULT_DOCKER_REGISTRY_MIRROR3}
)

func validateHostInfo(host apis.CloudHost) (err error) {
	log.Infof("Get hosts: %#v", host)
	if host.ManagerUrl == "" {
		err = fmt.Errorf("Host %q not found manager_uri", host.Id)
		return
	}

	if !utils.IsInStringArray(host.HostType, []string{cloudmodels.HOST_TYPE_HYPERVISOR, cloudmodels.HOST_TYPE_KUBELET}) {
		err = fmt.Errorf("Host type %q not support", host.HostType)
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

func validateCloudHostCondition(host apis.CloudHost) error {
	id := host.Id
	if id == "" {
		return httperrors.NewNotFoundError("Host %q not found", id)
	}

	err := validateHostInfo(host)
	if err != nil {
		httperrors.NewInputParameterError("Validate host %q info: %v", host.Name, err)
	}

	node, err := NodeManager.FetchNodeByHostId(id)
	if err != nil && err != sql.ErrNoRows {
		return httperrors.NewInternalServerError("Fetch node by host id %q: %v", id, err)
	}
	if node != nil {
		return httperrors.NewInputParameterError("Host %q already used by node %q", id, node.Name)
	}
	return nil
}

func nodeValidate(node yketypes.ConfigNode) (host apis.CloudHost, err error) {
	address := node.Address
	if address == "" {
		err = httperrors.NewInputParameterError("Empty YKE node config %q", node.NodeName)
		return
	}
	session, err := GetAdminSession()
	if err != nil {
		err = httperrors.NewInternalServerError("Get admin session: %v", err)
		return
	}
	params := jsonutils.NewDict()
	filter := jsonutils.NewArray()
	filter.Add(jsonutils.NewString(fmt.Sprintf("access_ip.equals(%s)", address)))
	params.Add(filter, "filter")
	ret, err := cloudmod.Hosts.List(session, params)
	if err != nil {
		return
	}
	if ret.Total == 0 {
		err = httperrors.NewInputParameterError("Not found cloud host by address: %s", address)
		return
	}
	if ret.Total > 1 {
		err = httperrors.NewInternalServerError("Duplicate cloud host fond by address: %s", address)
		return
	}
	host = apis.CloudHost{}
	err = ret.Data[0].Unmarshal(&host)
	if err != nil {
		return
	}
	err = validateCloudHostCondition(host)
	return
}

type YkeConfigNodeFactory struct {
	node       *SNode
	CreateData *jsonutils.JSONDict
}

func newNodeModelCreateData(
	cluster *SCluster,
	cloudHost apis.CloudHost,
	ykeNode yketypes.ConfigNode,
) *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(cluster.Id), "cluster")
	data.Add(jsonutils.NewStringArray(ykeNode.Role), "roles")
	data.Add(jsonutils.NewString(cloudHost.Id), "host")
	return data
}

func ModelCreateFields(man db.IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range man.TableSpec().Columns() {
		tags := col.Tags()
		create, _ := tags["create"]
		update := tags["update"]
		if update == "user" || (update == "admin" && userCred.IsSystemAdmin()) || create == "required" || create == "optional" || ((create == "admin_required" || create == "admin_optional") && userCred.IsSystemAdmin()) {
			ret = append(ret, col.Name())
		}
	}
	return ret
}

func NewYkeConfigNodeFactory(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ykeNode yketypes.ConfigNode,
	cluster *SCluster,
) (node *YkeConfigNodeFactory, err error) {
	cloudHost, err := nodeValidate(ykeNode)
	if err != nil {
		return
	}

	newData := newNodeModelCreateData(cluster, cloudHost, ykeNode)
	model, err := NewNode(ctx, userCred, newData)
	if err != nil {
		return
	}

	node = &YkeConfigNodeFactory{
		node:       model,
		CreateData: newData,
	}
	return
}

func (n *YkeConfigNodeFactory) Node() *SNode {
	return n.node
}

func (n *YkeConfigNodeFactory) Save() error {
	return NodeManager.TableSpec().Insert(n.node)
}

func allowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}
