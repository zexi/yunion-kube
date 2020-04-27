package tasks

import (
	"context"
	"fmt"
	"yunion.io/x/yunion-kube/pkg/models"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/apis"
)

func init() {
	taskman.RegisterTask(ComponentDeleteTask{})
}

type ComponentDeleteTask struct {
	taskman.STask
}

func (t *ComponentDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	comp := obj.(*models.SComponent)
	t.SetStage("OnUndeployComplete", nil)
	comp.StartComponentUndeployTask(ctx, t.UserCred, data.(*jsonutils.JSONDict), t.GetTaskId())
}

func (t *ComponentDeleteTask) OnUndeployComplete(ctx context.Context, obj *models.SComponent, data jsonutils.JSONObject) {
	obj.DeleteWithJoint(ctx, t.UserCred)
	t.SetStageComplete(ctx, nil)
}

func (t *ComponentDeleteTask) OnUndeployCompleteFailed(ctx context.Context, obj *models.SComponent, reason jsonutils.JSONObject) {
	t.onError(ctx, obj, fmt.Errorf("%s", reason))
}

func (t *ComponentDeleteTask) onError(ctx context.Context, obj *models.SComponent, err error) {
	reason := err.Error()
	obj.SetStatus(t.UserCred, apis.ComponentStatusDeleteFail, reason)
	t.STask.SetStageFailed(ctx, reason)
}
