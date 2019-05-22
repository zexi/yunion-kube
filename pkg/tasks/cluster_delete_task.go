package tasks

import (
	"context"
	"fmt"

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
	if err := t.startDeleteMachines(ctx, cluster); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
}

func (t *ClusterDeleteTask) startDeleteMachines(ctx context.Context, cluster *clusters.SCluster) error {
	ms, err := cluster.GetMachines()
	if err != nil {
		return fmt.Errorf("Get machines: %v", err)
	}
	if len(ms) == 0 {
		t.OnMachinesDeleted(ctx, cluster, nil)
		return nil
	}
	t.SetStage("OnMachinesDeleted", nil)
	return cluster.StartDeleteMachinesTask(ctx, t.GetUserCred(), ms, nil, t.GetTaskId())
}

func (t *ClusterDeleteTask) OnMachinesDeleted(ctx context.Context, cluster *clusters.SCluster, data jsonutils.JSONObject) {
	if err := cluster.RealDelete(ctx, t.UserCred); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterDeleteTask) OnMachinesDeletedFailed(ctx context.Context, cluster *clusters.SCluster, data jsonutils.JSONObject) {
	t.onError(ctx, cluster, fmt.Errorf("%s", data.String()))
}

func (t *ClusterDeleteTask) onError(ctx context.Context, cluster db.IStandaloneModel, err error) {
	t.SetFailed(ctx, cluster, err.Error())
}

func (t *ClusterDeleteTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason string) {
	cluster := obj.(*clusters.SCluster)
	cluster.SetStatus(t.UserCred, types.ClusterStatusDeleteFail, "")
	t.STask.SetStageFailed(ctx, reason)
}
