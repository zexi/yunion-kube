package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/models"
)

func init() {
	taskman.RegisterTask(ReleaseCreateTask{})
}

type ReleaseCreateTask struct {
	taskman.STask
}

func (t *ReleaseCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStage("OnDeployComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		release := obj.(*models.SRelease)
		if err := release.DoCreate(); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

func (t *ReleaseCreateTask) OnDeployComplete(ctx context.Context, obj *models.SRelease, data jsonutils.JSONObject) {
	obj.SetStatus(t.UserCred, apis.ReleaseStatusDeployed, "")
	t.SetStageComplete(ctx, nil)
}

func (t *ReleaseCreateTask) OnDeployCompleteFailed(ctx context.Context, obj *models.SRelease, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, apis.ReleaseStatusDeployFail, reason.String())
}
