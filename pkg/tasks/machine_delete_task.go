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
	taskman.RegisterTask(MachineDeleteTask{})
}

type MachineDeleteTask struct {
	taskman.STask
}

func (t *MachineDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*machines.SMachine)
	driver := machine.GetDriver()
	err := driver.PostDelete(machine)
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *MachineDeleteTask) OnError(ctx context.Context, machine *machines.SMachine, err error) {
	machine.SetStatus(t.UserCred, types.MachineStatusDeleteFail, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
