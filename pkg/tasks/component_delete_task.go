package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
)

func init() {
	taskman.RegisterTask(ComponentDeleteTask{})
}

type ComponentDeleteTask struct {
	taskman.STask
}

func (t *ComponentDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	comp := obj.(*clusters.SComponent)
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
	t.SetStageComplete(ctx, nil)
}

func (t *ComponentDeleteTask) onError(ctx context.Context, obj *clusters.SComponent, err error) {
	reason := err.Error()
	obj.SetStatus(t.UserCred, apis.ComponentStatusDeleteFail, reason)
	t.STask.SetStageFailed(ctx, reason)
}
