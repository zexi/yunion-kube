package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/models"
)

func init() {
	taskman.RegisterTask(ClusterResourceSyncTask{})
}

type ClusterResourceSyncTask struct {
	ClusterResourceBaseTask
}

func (t *ClusterResourceSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	resObj, _ := t.getModelManager(obj)
	resObj.SetStatus(t.UserCred, api.ClusterResourceStatusSyncing, "sync resource")
	t.SetStage("OnSyncComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		cAPI := models.GetClusterResAPI()
		if err := cAPI.PerformSyncResource(resObj, ctx, t.UserCred); err != nil {
			return nil, errors.Wrapf(err, "sync %s resource", resObj.LogPrefix())
		}
		return nil, nil
	})
}

func (t *ClusterResourceSyncTask) OnSyncComplete(ctx context.Context, obj models.IClusterModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterResourceSyncTask) OnSyncCompleteFailed(ctx context.Context, obj models.IClusterModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.ClusterResourceStatusSyncFail, reason.String())
}
