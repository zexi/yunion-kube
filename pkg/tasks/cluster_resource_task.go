package tasks

import (
	"context"
	"yunion.io/x/log"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/models"
)

func init() {
	for _, t := range []interface{}{
		ClusterResourceCreateTask{},
		ClusterResourceDeleteTask{},
	} {
		taskman.RegisterTask(t)
	}
}

type ClusterResourceBaseTask struct {
	taskman.STask
}

func (t *ClusterResourceBaseTask) getModelManager(obj db.IStandaloneModel) (models.IClusterModel, models.IClusterModelManager) {
	resObj := obj.(models.IClusterModel)
	resMan := resObj.GetModelManager().(models.IClusterModelManager)
	return resObj, resMan
}

type ClusterResourceCreateTask struct {
	ClusterResourceBaseTask
}

func (t *ClusterResourceCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	resObj, resMan := t.getModelManager(obj)
	resObj.SetStatus(t.UserCred, api.ClusterResourceStatusCreating, "create resource")
	t.SetStage("OnCreateComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		obj, err := models.CreateRemoteObject(ctx, t.UserCred, resMan, resObj, t.GetParams())
		if err != nil {
			log.Errorf("CreateRemoteObject error: %v", err)
			return nil, errors.Wrap(err, "CreateRemoteObject")
		}
		return jsonutils.Marshal(obj), nil
	})
}

func (t *ClusterResourceCreateTask) OnCreateComplete(ctx context.Context, obj models.IClusterModel, data jsonutils.JSONObject) {
	obj.SetStatus(t.UserCred, api.ClusterResourceStatusCreated, "create resource")
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterResourceCreateTask) OnCreateComplateFailed(ctx context.Context, obj models.IClusterModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.ClusterResourceStatusCreateFail, reason.String())
}

type ClusterResourceDeleteTask struct {
	ClusterResourceBaseTask
}

func (t *ClusterResourceDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	resObj, resMan := t.getModelManager(obj)
	resObj.SetStatus(t.UserCred, api.ClusterResourceStatusDeleting, "delete resource")
	t.SetStage("OnDeleteComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		err := models.DeleteRemoteObject(ctx, t.UserCred, resMan, resObj, t.Params)
		if err != nil {
			log.Errorf("DeleteRemoteObject error: %v", err)
			return nil, errors.Wrap(err, "DeleteRemoteObject")
		}
		return jsonutils.Marshal(obj), nil
	})
}

func (t *ClusterResourceDeleteTask) OnDeleteComplete(ctx context.Context, obj models.IClusterModel, data jsonutils.JSONObject) {
	if err := obj.RealDelete(ctx, t.UserCred); err != nil {
		SetObjectTaskFailed(ctx, t, obj, api.ClusterResourceStatusDeleteFail, err.Error())
	}
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterResourceDeleteTask) OnDeleteComplateFailed(ctx context.Context, obj models.IClusterModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.ClusterResourceStatusDeleteFail, reason.String())
}
