package app

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"

	"yunion.io/x/yunion-kube/pkg/models"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()
	apiPrefix := "/api"
	taskman.AddTaskHandler(apiPrefix, app)

	for _, man := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		db.UserCacheManager,
		db.TenantCacheManager,
		db.Metadata,
	} {
		db.RegisterModelManager(man)
	}

	for _, man := range []db.IModelManager{
		db.OpsLog.SetKeyword("kube_event", "kube_events"),
		models.ClusterManager,
		models.NodeManager,
	} {
		db.RegisterModelManager(man)
		handler := db.NewModelHandler(man)
		dispatcher.AddModelDispatcher(apiPrefix, app, handler)
	}
}
