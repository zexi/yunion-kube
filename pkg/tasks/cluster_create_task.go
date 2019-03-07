package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

func init() {
	taskman.RegisterTask(ClusterCreateTask{})
}

type ClusterCreateTask struct {
	taskman.STask
}

func (t *ClusterCreateTask) getMachines(cluster *clusters.SCluster) ([]*types.CreateMachineData, error) {
	params := t.GetParams()
	ret := []*types.CreateMachineData{}
	ms := []types.CreateMachineData{}
	if err := params.Unmarshal(&ms, "machines"); err != nil {
		return nil, err
	}
	for _, m := range ms {
		m.ClusterId = cluster.Id
		tmp := m
		ret = append(ret, &tmp)
	}
	return ret, nil
}

func (t *ClusterCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	machines, err := t.getMachines(cluster)
	if err != nil {
		t.onError(ctx, obj, err)
		return
	}
	if len(machines) == 0 {
		t.OnApplyAddonsComplete(ctx, obj, data)
		return
	}
	t.CreateMachines(ctx, cluster)
}

func (t *ClusterCreateTask) CreateMachines(ctx context.Context, cluster *clusters.SCluster) {
	machines, err := t.getMachines(cluster)
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	t.SetStage("OnMachinesCreated", nil)
	driver := cluster.GetDriver()
	if err := driver.CreateMachines(ctx, t.GetUserCred(), cluster, machines); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	ms, err := clusters.FetchMachinesByCreateData(cluster, machines)
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	if err := driver.RequestDeployMachines(ctx, t.GetUserCred(), cluster, ms, t); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	t.OnMachinesCreated(ctx, cluster)
}

func (t *ClusterCreateTask) OnMachinesCreated(ctx context.Context, cluster *clusters.SCluster) {
	log.Infof("Machines created")
	t.SetStage("OnApplyAddonsComplete", nil)
	cluster.StartApplyAddonsTask(ctx, t.GetUserCred(), t.GetParams(), t.GetTaskId())
}

func (t *ClusterCreateTask) OnApplyAddonsComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	t.SetStage("OnSyncStatus", nil)
	cluster.StartSyncStatus(ctx, t.UserCred, t.GetTaskId())
}

func (t *ClusterCreateTask) OnApplyAddonsCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data.String())
}

func (t *ClusterCreateTask) OnSyncStatus(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterCreateTask) OnSyncStatusFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data.String())
}

func (t *ClusterCreateTask) onError(ctx context.Context, cluster db.IStandaloneModel, err error) {
	t.SetFailed(ctx, cluster, err.Error())
}

func (t *ClusterCreateTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason string) {
	cluster := obj.(*clusters.SCluster)
	cluster.SetStatus(t.UserCred, types.ClusterStatusCreateFail, "")
	t.STask.SetStageFailed(ctx, reason)
}
