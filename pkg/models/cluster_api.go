package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
)

var (
	clusterResAPI IClusterResAPI
)

func init() {
	GetClusterResAPI()
}

func GetClusterResAPI() IClusterResAPI {
	if clusterResAPI == nil {
		clusterResAPI = newClusterResAPI()
	}
	return clusterResAPI
}

type IClusterResAPI interface {
	NamespaceScope() INamespaceResAPI

	// StartResourceSyncTask start sync cluster model resource task
	StartResourceSyncTask(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentId string) error
	// PerformSyncResource sync remote cluster resource to local
	PerformSyncResource(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error
}

type INamespaceResAPI interface {
}

type sClusterResAPI struct {
	namespaceScope INamespaceResAPI
}

func newClusterResAPI() IClusterResAPI {
	a := new(sClusterResAPI)
	a.namespaceScope = newNamespaceResAPI(a)
	return a
}

func (a sClusterResAPI) NamespaceScope() INamespaceResAPI {
	return a.namespaceScope
}

func (a sClusterResAPI) StartResourceSyncTask(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterResourceSyncTask", obj, userCred, data, parentId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New ClusterResourceSyncTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (a sClusterResAPI) PerformSyncResource(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	remoteObj, err := obj.GetRemoteObject()
	if err != nil {
		return errors.Wrap(err, "get remote object")
	}

	_, err = db.Update(obj, func() error {
		if err := obj.UpdateFromRemoteObject(ctx, userCred, remoteObj); err != nil {
			return errors.Wrap(err, "update from remote object")
		}
		// TODO: check if need SetExternalId
		return nil
	})
	return err
}

type sNamespaceResAPI struct {
	clusterResAPI IClusterResAPI
}

func newNamespaceResAPI(a IClusterResAPI) INamespaceResAPI {
	na := new(sNamespaceResAPI)
	na.clusterResAPI = a
	return na
}
