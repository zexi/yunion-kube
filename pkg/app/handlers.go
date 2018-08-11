package app

import (
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/pkg/appsrv"
	"github.com/yunionio/pkg/appsrv/dispatcher"

	"yunion.io/yunion-kube/pkg/k8s"
	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/resources/pod"
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

	for _, man := range []k8s.IK8sResourceManager{
		pod.PodManager,
	} {
		handler := k8s.NewK8sResourceHandler(man)
		k8s.AddK8sResourceDispatcher(apiPrefix, app, handler)
	}
}
