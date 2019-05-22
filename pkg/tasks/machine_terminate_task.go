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
	taskman.RegisterTask(MachineTerminateTask{})
}

type MachineTerminateTask struct {
	taskman.STask
}

func (t *MachineTerminateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*machines.SMachine)

	driver := machine.GetDriver()
	session, err := machines.MachineManager.GetSession()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	err = driver.TerminateResource(session, machine)
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	cluster, err := machine.GetCluster()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	cluster.StartSyncStatus(ctx, t.UserCred, "")
	t.SetStageComplete(ctx, nil)
}

func (t *MachineTerminateTask) OnError(ctx context.Context, machine *machines.SMachine, err error) {
	machine.SetStatus(t.UserCred, types.MachineStatusTerminateFail, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
