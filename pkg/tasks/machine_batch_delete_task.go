package tasks

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/machines"
)

func init() {
	taskman.RegisterTask(MachineBatchDeleteTask{})
}

type MachineBatchDeleteTask struct {
	taskman.STask
}

func (self *MachineBatchDeleteTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	self.SetStage("OnDeleteMachines", nil)
	for _, obj := range objs {
		if err := self.doDelete(ctx, obj.(*machines.SMachine)); err != nil {
			self.SetStageFailed(ctx, errors.Wrapf(err, "delete machine %s", obj.GetName()).Error())
			return
		}
	}
}

func (self *MachineBatchDeleteTask) doDelete(ctx context.Context, machine *machines.SMachine) error {
	return machine.StartTerminateTask(ctx, self.GetUserCred(), nil, self.GetTaskId())
}

func (self *MachineBatchDeleteTask) OnDeleteMachines(ctx context.Context, objs []db.IStandaloneModel, data *jsonutils.JSONDict) {
	for _, obj := range objs {
		if err := obj.(*machines.SMachine).RealDelete(ctx, self.GetUserCred()); err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("Delete machine %s error: %v", obj.GetName(), err))
			return
		}
	}
	self.SetStageComplete(ctx, nil)
}

func (self *MachineBatchDeleteTask) OnDeleteMachinesFailed(ctx context.Context, objs []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageFailed(ctx, data.String())
}
