package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

func init() {
	taskman.RegisterTask(ClusterBatchCreateTask{})
}

type ClusterBatchCreateTask struct {
	taskman.STask
}

func (t *ClusterBatchCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Infof("ClusterBatchCreateTask do nothing as for now")
	t.SetStageComplete(ctx, nil)
}
