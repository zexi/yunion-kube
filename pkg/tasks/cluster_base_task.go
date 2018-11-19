package tasks

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models"
)

type SClusterBaseTask struct {
	taskman.STask
}

func (t *SClusterBaseTask) SetFailed(ctx context.Context, cluster *models.SCluster, err error) {
	cluster.SetStatus(t.UserCred, models.CLUSTER_STATUS_ERROR, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
