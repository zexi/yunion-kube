package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models"
)

type SClusterBaseTask struct {
	taskman.STask
}

func (t *SClusterBaseTask) getCluster() *models.SCluster {
	obj := t.GetObject()
	return obj.(*models.SCluster)
}

func (t *SClusterBaseTask) SetFailed(ctx context.Context, cluster *models.SCluster, err error) {
	cluster.SetStatus(t.UserCred, models.CLUSTER_STATUS_ERROR, err.Error())
	t.SetStageFailed(ctx, err.Error())
}

type SClusterAgentBaseTask struct {
	SClusterBaseTask
}

func (t *SClusterAgentBaseTask) StartNodesAgent(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode, data jsonutils.JSONObject) {
	for _, node := range nodes {
		err := node.StartAgentStartTask(ctx, t.UserCred, nil, t.GetTaskId())
		if err != nil {
			log.Errorf("Start node %q agent task error: %v", node.Name, err)
		}
	}
}

func (t *SClusterAgentBaseTask) RestartNodesAgent(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode, data jsonutils.JSONObject) {
	for _, node := range nodes {
		err := node.StartAgentRestartTask(ctx, t.UserCred, nil, t.GetTaskId())
		if err != nil {
			log.Errorf("Restart node %q agent task error: %v", node.Name, err)
		}
	}
}

func (t *SClusterAgentBaseTask) StopNodesAgent(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode, data jsonutils.JSONObject) {
	for _, node := range nodes {
		err := node.StartAgentStopTask(ctx, t.UserCred, nil, t.GetTaskId())
		if err != nil {
			log.Errorf("Stop node %q agent task error: %v", node.Name, err)
		}
	}
}
