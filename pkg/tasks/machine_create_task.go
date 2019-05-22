package tasks

import (
	"context"
	"fmt"

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
	t.SetStage("OnMachinePrepared", nil)
	if err := machine.GetDriver().RequestPrepareMachine(ctx, t.UserCred, machine, t); err != nil {
		t.OnError(ctx, machine, err)
		return
	}
}

func (t *MachineCreateTask) OnMachinePrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	//machine := obj.(*machines.SMachine)
	t.SetStageComplete(ctx, nil)
}

func (t *MachineCreateTask) OnMachinePreparedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*machines.SMachine)
	t.OnError(ctx, machine, fmt.Errorf(data.String()))
}

func (t *MachineCreateTask) OnError(ctx context.Context, machine *machines.SMachine, err error) {
	machine.SetStatus(t.UserCred, types.MachineStatusCreateFail, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
