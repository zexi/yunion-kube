package tasks

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/controllers"
	"yunion.io/x/yunion-kube/pkg/models"
)

type ClusterDeleteNodesTask struct {
	SClusterAgentBaseTask
}

func init() {
	taskman.RegisterTask(ClusterDeleteNodesTask{})
}

func (t *ClusterDeleteNodesTask) getDeleteNodes() ([]*models.SNode, error) {
	nodeIds, err := t.GetParams().GetArray(models.NODES_DEPLOY_IDS_KEY)
	if err != nil {
		return nil, err
	}
	ret := make([]*models.SNode, len(nodeIds))
	for i, idObj := range nodeIds {
		id, _ := idObj.GetString()
		node, err := models.NodeManager.FetchNodeById(id)
		if err != nil {
			return nil, err
		}
		ret[i] = node
	}
	return ret, nil
}

func (t *ClusterDeleteNodesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	nodes, err := t.getDeleteNodes()
	if err != nil {
		t.SetFailed(ctx, cluster, err)
		return
	}
	t.SetStage("OnWaitNodesAgentStart", nil)
	t.StartNodesAgent(ctx, cluster, nodes, data)
}

func (t *ClusterDeleteNodesTask) OnWaitNodesAgentStart(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	nodes, err := t.getDeleteNodes()
	if err != nil {
		t.SetFailed(ctx, cluster, fmt.Errorf("Get delete nodes: %v", err))
		return
	}
	if !cluster.IsNodesReady(nodes...) {
		log.Infof("Not all node ready, wait agents to start")
		time.Sleep(time.Second * 2)
		t.ScheduleRun(nil)
		return
	}
	log.Infof("All nodes agent started, start delete")
	err = t.doDelete(ctx, cluster, nodes)
	if err != nil {
		t.SetFailed(ctx, cluster, fmt.Errorf("do delete: %v", err))
		models.SetNodesStatus(nodes, models.NODE_STATUS_ERROR)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterDeleteNodesTask) doDelete(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode) error {
	models.SetNodesStatus(nodes, models.NODE_STATUS_DELETING)
	config, err := cluster.GetYKEConfig()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		config = models.RemoveYKEConfigNode(config, node)
	}
	if len(config.Nodes) == 0 {
		err = controllers.Manager.RemoveController(cluster)
		if err != nil {
			return fmt.Errorf("Remove cluster controller error: %v", err)
		}
		// do yke remove
		err = cluster.RemoveCluster(ctx)
		if err != nil {
			return fmt.Errorf("Cleanup cluster error: %v", err)
		}
	} else {
		if err != nil {
			return err
		}
		err = cluster.SetYKEConfig(config)
		if err != nil {
			log.Errorf("Set YKEConfig error: %v", err)
			return err
		}
		err = cluster.SyncUpdate(ctx)
		if err != nil {
			return err
		}
	}

	for _, node := range nodes {
		err = removeCloudContainers(node)
		if err != nil {
			return err
		}
	}

	for _, node := range nodes {
		err = node.RealDelete(ctx, t.UserCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeCloudContainers(node *models.SNode) error {
	session, err := models.GetAdminSession()
	if err != nil {
		return err
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(2000), "limit")
	params.Add(jsonutils.JSONTrue, "admin")
	params.Add(jsonutils.NewString("container"), "hypervisor")
	params.Add(jsonutils.NewString(node.Name), "host")
	result, err := cloudmod.Servers.List(session, params)
	if err != nil {
		return err
	}
	srvIds := []string{}
	for _, srv := range result.Data {
		id, _ := srv.GetString("id")
		srvIds = append(srvIds, id)
	}
	params = jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "override_pending_delete")
	cloudmod.Servers.BatchDeleteWithParam(session, srvIds, params, nil)
	return nil
}
