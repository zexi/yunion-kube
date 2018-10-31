package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models"
)

type SNodeBaseTask struct {
	taskman.STask
}

func (t *SNodeBaseTask) getNode() *models.SNode {
	obj := t.GetObject()
	return obj.(*models.SNode)
}

func (t *SNodeBaseTask) onFailed(ctx context.Context, obj db.IStandaloneModel, errStr string) {
	node := obj.(*models.SNode)
	node.SetStatus(t.UserCred, models.NODE_STATUS_ERROR, errStr)
	t.SetStageFailed(ctx, errStr)
}

func (t *SNodeBaseTask) OnFailed(ctx context.Context, obj db.IStandaloneModel, err error) {
	t.onFailed(ctx, obj, err.Error())
}

func (t *SNodeBaseTask) OnFailedJson(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.onFailed(ctx, obj, data.String())
}
