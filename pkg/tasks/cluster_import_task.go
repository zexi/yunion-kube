package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
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
	nodes, err := cluster.GetNodes()
	if err != nil {
		t.SetFailed(ctx, cluster, err)
		return
	}
	t.StartDeployCluster(ctx, cluster, nodes)
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
