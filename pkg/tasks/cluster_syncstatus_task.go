package tasks

import (
	"context"

	"yunion.io/x/yunion-kube/pkg/models"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/utils/logclient"
)

func init() {
	taskman.RegisterTask(ClusterSyncstatusTask{})
}

type ClusterSyncstatusTask struct {
	taskman.STask
}

func (t *ClusterSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	mCnt, err := cluster.GetMachinesCount()
	if err != nil {
		t.onError(ctx, cluster, err.Error())
		return
	}
	if mCnt == 0 && cluster.GetDriver().NeedCreateMachines() {
		cluster.SetStatus(t.UserCred, apis.ClusterStatusInit, "")
		t.SetStageComplete(ctx, nil)
		return
	}

	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		k8sCli, err := cluster.GetK8sClient()
		if err != nil {
			return nil, err
		}
		info, err := k8sCli.Discovery().ServerVersion()
		if err != nil {
			return nil, err
		}
		log.Infof("Get %s cluster k8s version: %#v", cluster.GetName(), info)
		cluster.SetK8sVersion(info.String())
		return nil, nil
	})
}

func (t *ClusterSyncstatusTask) OnSyncStatus(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	cluster.SetStatus(t.UserCred, apis.ClusterStatusRunning, "")
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterSyncstatusTask) OnSyncStatusFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.onError(ctx, cluster, data.String())
}

func (t *ClusterSyncstatusTask) onError(ctx context.Context, cluster db.IStandaloneModel, err string) {
	t.SetFailed(ctx, cluster, err)
	logclient.AddActionLogWithStartable(t, cluster, logclient.ActionClusterSyncStatus, err, t.UserCred, false)
}

func (t *ClusterSyncstatusTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason string) {
	cluster := obj.(*models.SCluster)
	cluster.SetStatus(t.UserCred, apis.ClusterStatusUnknown, "")
	t.STask.SetStageFailed(ctx, reason)
}
