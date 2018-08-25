package tasks

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models"
)

type ClusterDeployTask struct {
	SClusterAgentBaseTask
}

func init() {
	taskman.RegisterTask(ClusterDeployTask{})
}

func (t *ClusterDeployTask) getPendingDeployNodes() ([]*models.SNode, error) {
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

func (t *ClusterDeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	nodes, err := t.getPendingDeployNodes()
	if err != nil {
		t.SetFailed(ctx, cluster, err)
		return
	}
	t.SetStage("OnWaitNodesAgentStart", nil)
	t.StartNodesAgent(ctx, cluster, nodes, data)
}

func (t *ClusterDeployTask) OnWaitNodesAgentStart(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	nodes, _ := cluster.GetNodes()
	if !cluster.IsNodesReady(nodes...) {
		log.Infof("Not all node ready, wait agents to start")
		time.Sleep(time.Second * 2)
		t.ScheduleRun(nil)
		return
	}
	log.Infof("All nodes agent started, start deploy")
	t.doDeploy(ctx, cluster, nodes)
}

func (t *ClusterDeployTask) OnWaitNodesAgentStartFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("============callback failed: %s", data)
	t.SetFailed(ctx, obj.(*models.SCluster), fmt.Errorf("OnWaitNodesAgentStart: %s", data))
}

func (t *ClusterDeployTask) doDeploy(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode) {
	cluster.Deploy(ctx, nodes...)
	t.SetStageComplete(ctx, nil)
}
