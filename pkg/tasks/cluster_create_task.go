package tasks

import (
	"context"
	"fmt"
	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/utils/logclient"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

func init() {
	taskman.RegisterTask(ClusterCreateTask{})
}

type ClusterCreateTask struct {
	taskman.STask
}

func (t *ClusterCreateTask) getMachines(cluster *models.SCluster) ([]*api.CreateMachineData, error) {
	if !cluster.GetDriver().NeedCreateMachines() {
		return nil, nil
	}
	params := t.GetParams()
	ret := []*api.CreateMachineData{}
	ms := []api.CreateMachineData{}
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
	cluster := obj.(*models.SCluster)
	machines, err := t.getMachines(cluster)
	if err != nil {
		t.onError(ctx, obj, err)
		return
	}
	if len(machines) == 0 {
		t.OnApplyAddonsComplete(ctx, obj, data)
		return
	}
	res := api.ClusterCreateInput{}
	if err := t.GetParams().Unmarshal(&res); err != nil {
		t.onError(ctx, obj, fmt.Errorf("Unmarshal: %v", err))
		return
	}
	// generate certificates if needed
	if err := cluster.GenerateCertificates(ctx, t.GetUserCred()); err != nil {
		t.onError(ctx, obj, fmt.Errorf("GenerateCertificates: %v", err))
		return
	}
	t.CreateMachines(ctx, cluster)
}

func (t *ClusterCreateTask) CreateMachines(ctx context.Context, cluster *models.SCluster) {
	machines, err := t.getMachines(cluster)
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	t.SetStage("OnMachinesCreated", nil)
	cluster.StartCreateMachinesTask(ctx, t.GetUserCred(), machines, t.GetTaskId())
}

func (t *ClusterCreateTask) OnMachinesCreated(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetStage("OnApplyAddonsComplete", nil)
	cluster.StartApplyAddonsTask(ctx, t.GetUserCred(), t.GetParams(), t.GetTaskId())
	logclient.AddActionLogWithStartable(t, cluster, logclient.ActionClusterAddMachine, nil, t.UserCred, true)
}

func (t *ClusterCreateTask) OnMachinesCreatedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data)
}

func (t *ClusterCreateTask) OnApplyAddonsComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	t.SetStage("OnSyncStatus", nil)
	cluster.StartSyncStatus(ctx, t.UserCred, t.GetTaskId())
	logclient.AddActionLogWithStartable(t, obj, logclient.ActionClusterApplyAddons, nil, t.UserCred, true)
}

func (t *ClusterCreateTask) OnApplyAddonsCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data)
}

func (t *ClusterCreateTask) OnSyncStatus(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
	logclient.AddActionLogWithStartable(t, obj, logclient.ActionClusterSyncStatus, nil, t.UserCred, true)
}

func (t *ClusterCreateTask) OnSyncStatusFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data)
}

func (t *ClusterCreateTask) onError(ctx context.Context, cluster db.IStandaloneModel, err error) {
	t.SetFailed(ctx, cluster, jsonutils.NewString(err.Error()))
}

func (t *ClusterCreateTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	cluster.SetStatus(t.UserCred, api.ClusterStatusCreateFail, reason.String())
	t.STask.SetStageFailed(ctx, reason)
	logclient.AddActionLogWithStartable(t, obj, logclient.ActionClusterCreate, reason, t.UserCred, false)
}
