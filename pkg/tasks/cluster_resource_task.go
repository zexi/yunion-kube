package tasks

import (
	"context"
	"yunion.io/x/log"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
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
	resObj.SetStatus(t.UserCred, apis.ClusterResourceStatusCreating, "create resource")
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
	obj.SetStatus(t.UserCred, apis.ClusterResourceStatusCreated, "create resource")
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterResourceCreateTask) OnCreateComplateFailed(ctx context.Context, obj models.IClusterModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, apis.ClusterResourceStatusCreateFail, reason.String())
}

type ClusterResourceDeleteTask struct {
	ClusterResourceBaseTask
}

func (t *ClusterResourceDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	resObj, resMan := t.getModelManager(obj)
	resObj.SetStatus(t.UserCred, apis.ClusterResourceStatusDeleting, "delete resource")
	t.SetStage("OnDeleteComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		err := models.DeleteRemoteObject(ctx, t.UserCred, resMan, resObj, t.Params)
		if err != nil {
			log.Errorf("DeleteRemoteObject error: %v", err)
			return nil, errors.Wrap(err, "DeleteRemoteObject")
		}
		if err := resObj.RealDelete(ctx, t.UserCred); err != nil {
			return nil, errors.Wrap(err, "RealDelete")
		}
		return jsonutils.Marshal(obj), nil
	})
}

func (t *ClusterResourceDeleteTask) OnDeleteComplete(ctx context.Context, obj models.IClusterModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterResourceDeleteTask) OnDeleteComplateFailed(ctx context.Context, obj models.IClusterModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, apis.ClusterResourceStatusDeleteFail, reason.String())
}
