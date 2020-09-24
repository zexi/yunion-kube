package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/utils/logclient"
)

func init() {
	taskman.RegisterTask(ClusterSyncTask{})
}

type ClusterSyncTask struct {
	taskman.STask
}

func (t *ClusterSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	t.SetStage("OnSyncComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		if err := client.GetClustersManager().AddClient(cluster); err != nil {
			if errors.Cause(err) != client.ErrClusterAlreadyAdded {
				return nil, errors.Wrap(err, "add cluster to client manager")
			}
		}
		// do sync
		if err := cluster.SyncCallSyncTask(ctx, t.UserCred); err != nil {
			return nil, errors.Wrap(err, "SyncCallSyncTask")
		}
		return nil, nil
	})
}

func (t *ClusterSyncTask) OnSyncComplete(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterSync, nil, t.UserCred, true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterSyncTask) OnSyncCompleteFailed(ctx context.Context, cluster *models.SCluster, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterSync, reason, t.UserCred, false)
}
