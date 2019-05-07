package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type ClusterDeleteMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterDeleteMachinesTask{})
}

func (t *ClusterDeleteMachinesTask) getDeleteMachines() ([]manager.IMachine, error) {
	machinesData, err := t.GetParams().GetArray("machines")
	if err != nil {
		return nil, err
	}
	machines := []manager.IMachine{}
	for _, obj := range machinesData {
		id, err := obj.GetString()
		if err != nil {
			return nil, err
		}
		machineObj, err := manager.MachineManager().FetchMachineByIdOrName(t.UserCred, id)
		if err != nil {
			return nil, err
		}
		machines = append(machines, machineObj)
	}
	return machines, nil
}

func (t *ClusterDeleteMachinesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	ms, err := t.getDeleteMachines()
	if err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	t.SetStage("OnDeleteMachines", nil)
	if err := cluster.GetDriver().RequestDeleteMachines(ctx, t.UserCred, cluster, ms, t); err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
}

func (t *ClusterDeleteMachinesTask) OnDeleteMachines(ctx context.Context, cluster *clusters.SCluster, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterDeleteMachinesTask) OnDeleteMachinesFailed(ctx context.Context, cluster *clusters.SCluster, data jsonutils.JSONObject) {
	t.OnError(ctx, cluster, fmt.Errorf(data.String()))
}

func (t *ClusterDeleteMachinesTask) OnError(ctx context.Context, cluster *clusters.SCluster, err error) {
	cluster.SetStatus(t.UserCred, types.ClusterStatusError, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
