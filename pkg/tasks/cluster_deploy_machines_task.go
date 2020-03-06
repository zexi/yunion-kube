package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/manager"
)

type ClusterDeployMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterDeployMachinesTask{})
}

func (t *ClusterDeployMachinesTask) getAddMachines() ([]manager.IMachine, error) {
	msIds := []string{}
	if err := t.Params.Unmarshal(&msIds, clusters.MachinesDeployIdsKey); err != nil {
		return nil, err
	}
	ms := make([]manager.IMachine, 0)
	for _, id := range msIds {
		m, err := machines.MachineManager.FetchMachineById(id)
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}

func (t *ClusterDeployMachinesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	ms, err := t.getAddMachines()
	if err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	if err := cluster.GetDriver().RequestDeployMachines(ctx, t.UserCred, cluster, ms, t); err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	cluster.StartApplyAddonsTask(ctx, t.UserCred, nil, "")
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterDeployMachinesTask) OnError(ctx context.Context, cluster *clusters.SCluster, err error) {
	cluster.SetStatus(t.UserCred, apis.ClusterStatusError, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
