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

type ClusterImportTask struct {
	SClusterBaseTask
}

func init() {
	taskman.RegisterTask(ClusterImportTask{})
}

func (t *ClusterImportTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	cluster.SetStatus(t.UserCred, models.CLUSTER_STATUS_IMPORT, "")
	t.StartNodesAgent(ctx, cluster, data)
}

func (t *ClusterImportTask) fetchNodeConfigData(nodeId string) (*jsonutils.JSONDict, error) {
	data := t.GetParams()
	nodeConfigs, err := data.GetMap(models.NODES_CONFIG_DATA_KEY)
	if err != nil {
		return nil, err
	}
	config, ok := nodeConfigs[nodeId]
	if !ok {
		return nil, fmt.Errorf("Can't get node %q import config", nodeId)
	}
	return config.(*jsonutils.JSONDict), nil
}

func (t *ClusterImportTask) StartNodesAgent(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	nodes, err := cluster.GetNodes()
	if err != nil {
		t.SetFailed(ctx, cluster, err)
		return
	}
	for _, node := range nodes {
		config, err := t.fetchNodeConfigData(node.Id)
		if err != nil {
			t.SetFailed(ctx, cluster, err)
			return
		}
		node.StartAgentStartTask(ctx, t.UserCred, config, t.GetTaskId())
	}
	t.SetStage("OnWaitNodesAgentStart", nil)
}

func (t *ClusterImportTask) OnWaitNodesAgentStart(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	nodes, _ := cluster.GetNodes()
	if !cluster.IsNodesReady(nodes...) {
		log.Infof("Not all node ready, wait agents to start")
		time.Sleep(time.Second * 2)
		t.ScheduleRun(nil)
		return
	}
	log.Infof("All nodes agent started, start deploy")
	t.StartDeployCluster(ctx, cluster, nodes)
}

func (t *ClusterImportTask) OnWaitNodesAgentStartFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("============callback failed: %s", data)
	t.SetFailed(ctx, obj.(*models.SCluster), fmt.Errorf("OnWaitNodesAgentStart: %s", data))
}

func (t *ClusterImportTask) StartDeployCluster(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode) {
	t.SetStage("OnDeployComplete", nil)
	cluster.StartClusterDeployTask(ctx, t.UserCred, models.FetchClusterDeployTaskData(nodes), t.GetTaskId())
}

func (t *ClusterImportTask) OnDeployComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterImportTask) OnDeployCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj.(*models.SCluster), fmt.Errorf("OnDeployCompleteFailed: %s", data))
}
