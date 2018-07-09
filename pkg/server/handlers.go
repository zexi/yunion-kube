package server

import (
	"yunion.io/yunioncloud/pkg/appsrv"
	"yunion.io/yunioncloud/pkg/appsrv/dispatcher"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/cloudcommon/db/taskman"

	"yunion.io/yunion-kube/pkg/models"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()
	taskman.AddTaskHandler("/api", app)

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
		models.ClusterManager,
		models.NodeManager,
	} {
		db.RegisterModelManager(man)
		handler := db.NewModelHandler(man)
		dispatcher.AddModelDispatcher("/api", app, handler)
	}
}
