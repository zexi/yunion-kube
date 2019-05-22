package tasks

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	//"yunion.io/x/yunion-kube/pkg/models/clusters"
	//"yunion.io/x/yunion-kube/pkg/models/machines"
)

type IStatusStandaloneModel interface {
	db.IStandaloneModel

	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
}

func SetObjectTaskFailed(ctx context.Context, task taskman.ITask, obj IStatusStandaloneModel, status, reason string) {
	if len(status) > 0 {
		obj.SetStatus(task.GetUserCred(), status, reason)
	}
	task.SetStageFailed(ctx, reason)
}
