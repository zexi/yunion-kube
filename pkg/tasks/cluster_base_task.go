package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
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

func (t *SClusterAgentBaseTask) StartNodesAgent(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode, data jsonutils.JSONObject) error {
	for _, node := range nodes {
		if cluster.IsNodeAgentReady(node) {
			continue
		}
		err := node.StartAgentStartTask(ctx, t.UserCred, nil, t.GetTaskId())
		if err != nil {
			err = fmt.Errorf("Start node %q agent task error: %v", node.Name, err)
			return err
		}
	}
	return nil
}

func (t *SClusterAgentBaseTask) RestartNodesAgent(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode, data jsonutils.JSONObject) error {
	for _, node := range nodes {
		err := node.StartAgentRestartTask(ctx, t.UserCred, nil, t.GetTaskId())
		if err != nil {
			err = fmt.Errorf("Restart node %q agent task error: %v", node.Name, err)
			return err
		}
	}
	return nil
}

func (t *SClusterAgentBaseTask) StopNodesAgent(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode, data jsonutils.JSONObject) error {
	for _, node := range nodes {
		err := node.StartAgentStopTask(ctx, t.UserCred, nil, t.GetTaskId())
		if err != nil {
			err = fmt.Errorf("Stop node %q agent task error: %v", node.Name, err)
			return err
		}
	}
	return nil
}
