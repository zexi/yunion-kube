package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type ClusterAddMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterAddMachinesTask{})
}

func (t *ClusterAddMachinesTask) getAddMachines() ([]*types.CreateMachineData, error) {
	ms := []types.CreateMachineData{}
	if err := t.Params.Unmarshal(&ms, "machines"); err != nil {
		return nil, err
	}
	machines := make([]*types.CreateMachineData, len(ms))
	for i := range ms {
		machines[i] = &ms[i]
	}
	return machines, nil
}

func (t *ClusterAddMachinesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	ms, err := t.getAddMachines()
	if err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	if err := cluster.GetDriver().RequestCreateMachines(ctx, t.UserCred, cluster, ms, t); err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	cluster.StartApplyAddonsTask(ctx, t.UserCred, nil, "")
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterAddMachinesTask) OnError(ctx context.Context, cluster *clusters.SCluster, err error) {
	cluster.SetStatus(t.UserCred, types.ClusterStatusError, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
