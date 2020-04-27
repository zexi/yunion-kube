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
	taskman.RegisterTask(ComponentDeployTask{})
}

type ComponentDeployTask struct {
	taskman.STask
}

func (t *ComponentDeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	comp := obj.(*models.SComponent)
	cluster, err := comp.GetCluster()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	t.SetStage("OnDeployComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		drv, err := comp.GetDriver()
		if err != nil {
			return nil, err
		}
		settings, err := comp.GetSettings()
		if err != nil {
			return nil, err
		}
		if err := drv.DoEnable(cluster, settings); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

func (t *ComponentDeployTask) OnDeployComplete(ctx context.Context, obj *models.SComponent, data jsonutils.JSONObject) {
	obj.SetStatus(t.UserCred, apis.ComponentStatusDeployed, "")
	if err := obj.DeleteWithJoint(ctx, t.UserCred); err != nil {
		t.onError(ctx, obj, err)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *ComponentDeployTask) OnDeployCompleteFailed(ctx context.Context, obj *models.SComponent, reason jsonutils.JSONObject) {
	t.onError(ctx, obj, fmt.Errorf(reason.String()))
}

func (t *ComponentDeployTask) onError(ctx context.Context, obj *models.SComponent, err error) {
	reason := err.Error()
	obj.SetStatus(t.UserCred, apis.ComponentStatusDeployFail, reason)
	t.STask.SetStageFailed(ctx, reason)
}
