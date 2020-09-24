package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/utils/logclient"
)

func init() {
	for _, t := range []interface{}{
		FedResourceUpdateTask{},
		FedResourceSyncTask{},
	} {
		taskman.RegisterTask(t)
	}
}

type FedResourceBaseTask struct {
	taskman.STask
}

type FedResourceUpdateTask struct {
	FedResourceBaseTask
}

func (t *FedResourceUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStage("OnUpdateComplete", nil)
	fedApi := models.GetFedResAPI()
	fedObj := obj.(models.IFedModel)
	if err := fedApi.StartSyncTask(fedObj, ctx, t.GetUserCred(), t.GetParams(), t.GetTaskId()); err != nil {
		t.OnUpdateCompleteFailed(ctx, fedObj, jsonutils.NewString(err.Error()))
	}
}

func (t *FedResourceUpdateTask) OnUpdateComplete(ctx context.Context, obj models.IFedModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
	logclient.LogWithStartable(t, obj, logclient.ActionResourceUpdate, nil, t.GetUserCred(), true)
}

func (t *FedResourceUpdateTask) OnUpdateCompleteFailed(ctx context.Context, obj models.IFedModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.FederatedResourceStatusUpdateFail, reason.String())
	logclient.LogWithStartable(t, obj, logclient.ActionResourceUpdate, reason, t.GetUserCred(), false)
}

type FedResourceSyncTask struct {
	FedResourceBaseTask
}

func (t *FedResourceSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStage("OnSyncComplete", nil)
	fedApi := models.GetFedResAPI()
	fedObj := obj.(models.IFedModel)
	fedObj.SetStatus(t.GetUserCred(), api.FederatedResourceStatusSyncing, "start syncing")
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		clusters, err := fedApi.GetAttachedClusters(fedObj)
		if err != nil {
			return nil, errors.Wrap(err, "get attached clusters")
		}
		for _, cluster := range clusters {
			input := api.FederatedResourceJointClusterInput{
				ClusterId: cluster.GetId(),
			}
			if err := fedApi.PerformSyncCluster(fedObj, ctx, t.UserCred, input.JSON(input)); err != nil {
				log.Errorf("%s sync to cluster %s(%s) error: %v", fedObj.LogPrefix(), cluster.GetName(), cluster.GetId(), err)
			}
		}
		return nil, nil
	})
}

func (t *FedResourceSyncTask) OnSyncComplete(ctx context.Context, obj models.IFedModel, data jsonutils.JSONObject) {
	obj.SetStatus(t.GetUserCred(), api.FederatedResourceStatusActive, "sync complete")
	t.SetStageComplete(ctx, nil)
	logclient.LogWithStartable(t, obj, logclient.ActionResourceSync, nil, t.GetUserCred(), true)
}

func (t *FedResourceSyncTask) OnSyncCompleteFailed(ctx context.Context, obj models.IFedModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.FedreatedResourceStatusSyncFail, reason.String())
	logclient.LogWithStartable(t, obj, logclient.ActionResourceSync, reason, t.GetUserCred(), false)
}
