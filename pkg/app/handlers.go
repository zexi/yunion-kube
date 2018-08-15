package app

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/k8s"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/resources/configmap"
	"yunion.io/x/yunion-kube/pkg/resources/deployment"
	//"yunion.io/x/yunion-kube/pkg/resources/namespace"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	"yunion.io/x/yunion-kube/pkg/resources/service"
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
		configmap.ConfigMapManager,
		deployment.DeploymentManager,
		pod.PodManager,
		service.ServiceManager,
		//namespace.NamespaceManager,
	} {
		handler := k8s.NewK8sResourceHandler(man)
		k8s.AddResourceDispatcher(apiPrefix, app, handler)
	}

	k8s.AddMiscDispatcher(apiPrefix, app)
}
