package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

func init() {
	taskman.RegisterTask(ClusterDeleteTask{})
}

type ClusterDeleteTask struct {
	taskman.STask
}

func (t *ClusterDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	t.startDeleteMachines(ctx, cluster)
}

func (t *ClusterDeleteTask) startDeleteMachines(ctx context.Context, cluster *clusters.SCluster) {
	if err := cluster.DeleteMachines(ctx, t.UserCred); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	if err := cluster.GetDriver().RequestDeleteCluster(cluster); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	if err := cluster.RealDelete(ctx, t.UserCred); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterDeleteTask) onError(ctx context.Context, cluster db.IStandaloneModel, err error) {
	t.SetFailed(ctx, cluster, err.Error())
}

func (t *ClusterDeleteTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason string) {
	cluster := obj.(*clusters.SCluster)
	cluster.SetStatus(t.UserCred, types.ClusterStatusDeleteFail, "")
	t.STask.SetStageFailed(ctx, reason)
}
