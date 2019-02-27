package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

func init() {
	taskman.RegisterTask(MachinePrepareTask{})
}

type MachinePrepareTask struct {
	taskman.STask
}

func (t *MachinePrepareTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*machines.SMachine)
	param := t.GetParams()

	prepareData := machines.MachinePrepareData{}
	if err := param.Unmarshal(&prepareData); err != nil {
		t.OnError(ctx, machine, err)
		return
	}

	prepareData.InstanceId = machine.ResourceId
	driver := machine.GetDriver()
	session, err := machines.MachineManager.GetSession()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	_, err = driver.PrepareResource(session, machine, &prepareData)
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	machine.SetStatus(t.UserCred, types.MachineStatusRunning, "")

	log.Infof("Prepare machine complete")
	cluster, err := machine.GetCluster()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	cluster.StartSyncStatus(ctx, t.UserCred, "")
	t.SetStageComplete(ctx, nil)
}

func (t *MachinePrepareTask) OnError(ctx context.Context, machine *machines.SMachine, err error) {
	machine.SetStatus(t.UserCred, types.MachineStatusPrepareFail, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
