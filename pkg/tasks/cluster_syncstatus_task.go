package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/utils/logclient"
)

func init() {
	taskman.RegisterTask(ClusterSyncstatusTask{})
}

type ClusterSyncstatusTask struct {
	taskman.STask
}

func (t *ClusterSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	mCnt, err := cluster.GetMachinesCount()
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	if mCnt == 0 && cluster.GetDriver().NeedCreateMachines() {
		cluster.SetStatus(t.UserCred, apis.ClusterStatusInit, "")
		t.SetStageComplete(ctx, nil)
		return
	}

	k8sCli, err := cluster.GetK8sClient()
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	info, err := k8sCli.Discovery().ServerVersion()
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	log.Infof("Get %s cluster k8s version: %#v", cluster.GetName(), info)
	cluster.SetStatus(t.UserCred, apis.ClusterStatusRunning, "")
	cluster.SetK8sVersion(info.String())
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterSyncstatusTask) onError(ctx context.Context, cluster db.IStandaloneModel, err error) {
	t.SetFailed(ctx, cluster, err.Error())
	logclient.AddActionLogWithStartable(t, cluster, logclient.ActionClusterSyncStatus, err.Error(), t.UserCred, false)
}

func (t *ClusterSyncstatusTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason string) {
	cluster := obj.(*clusters.SCluster)
	cluster.SetStatus(t.UserCred, apis.ClusterStatusUnknown, "")
	t.STask.SetStageFailed(ctx, reason)
}
