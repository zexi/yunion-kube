package tasks

import (
	"context"
	"yunion.io/x/yunion-kube/pkg/models"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/apis"
)

func init() {
	taskman.RegisterTask(ComponentUndeployTask{})
}

type ComponentUndeployTask struct {
	taskman.STask
}

func (t *ComponentUndeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	comp := obj.(*models.SComponent)
	cluster, err := comp.GetCluster()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	drv, err := comp.GetDriver()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	settings, err := comp.GetSettings()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	if err := drv.DoDisable(cluster, settings); err != nil {
		t.onError(ctx, comp, err)
		return
	}
	comp.SetStatus(t.UserCred, apis.ComponentStatusInit, "")
	comp.SetEnabled(false)
	t.SetStageComplete(ctx, nil)
}

func (t *ComponentUndeployTask) onError(ctx context.Context, obj *models.SComponent, err error) {
	reason := err.Error()
	obj.SetStatus(t.UserCred, apis.ComponentStatusUndeployFail, reason)
	t.STask.SetStageFailed(ctx, reason)
}
