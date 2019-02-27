package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

func init() {
	taskman.RegisterTask(MachineCreateTask{})
}

type MachineCreateTask struct {
	taskman.STask
}

func (t *MachineCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*machines.SMachine)
	cluster, err := machine.GetCluster()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	if err := machine.GetDriver().PostCreate(ctx, t.UserCred, cluster, machine, t.GetParams()); err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *MachineCreateTask) OnError(ctx context.Context, machine *machines.SMachine, err error) {
	machine.SetStatus(t.UserCred, types.MachineStatusCreateFail, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
