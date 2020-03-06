package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/utils/logclient"
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
	logclient.AddActionLogWithStartable(t, obj, logclient.ActionMachineCreate, nil, t.UserCred, true)
}

func (t *MachineCreateTask) OnMachinePreparedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*machines.SMachine)
	t.OnError(ctx, machine, fmt.Errorf(data.String()))
}

func (t *MachineCreateTask) OnError(ctx context.Context, machine *machines.SMachine, err error) {
	machine.SetStatus(t.UserCred, apis.MachineStatusCreateFail, err.Error())
	t.SetStageFailed(ctx, err.Error())
	logclient.AddActionLogWithStartable(t, machine, logclient.ActionMachineCreate, err.Error(), t.UserCred, false)
}
