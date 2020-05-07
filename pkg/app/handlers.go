package app

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/k8s"
	k8sdispatcher "yunion.io/x/yunion-kube/pkg/k8s/dispatcher"
	"yunion.io/x/yunion-kube/pkg/k8smodels"
	"yunion.io/x/yunion-kube/pkg/models"

	_ "yunion.io/x/yunion-kube/pkg/drivers/machines"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	_ "yunion.io/x/yunion-kube/pkg/k8smodels/drivers/secret"
	_ "yunion.io/x/yunion-kube/pkg/k8smodels/drivers/storageclass"
	_ "yunion.io/x/yunion-kube/pkg/models/drivers/release"
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
		db.SharedResourceManager,
		db.Metadata,
	} {
		db.RegisterModelManager(man)
	}

	for _, man := range []db.IModelManager{
		db.OpsLog,
		models.RepoManager,
		models.ClusterManager,
		models.X509KeyPairManager,
		models.ComponentManager,
		models.MachineManager,
		models.ReleaseManager,
	} {
		db.RegisterModelManager(man)
		handler := db.NewModelHandler(man)
		dispatcher.AddModelDispatcher(apiPrefix, app, handler)
	}

	for _, man := range []db.IJointModelManager{
		models.ClusterX509KeyPairManager,
		models.ClusterComponentManager,
	} {
		db.RegisterModelManager(man)
		handler := db.NewJointModelHandler(man)
		dispatcher.AddJointModelDispatcher(apiPrefix, app, handler)
	}

	for _, man := range []k8s.IK8sResourceManager{
		//k8sapp.AppFromFileManager,
		//release.ReleaseManager,
	} {
		handler := k8s.NewK8sResourceHandler(man)
		k8s.AddResourceDispatcher(apiPrefix, app, handler)
	}

	// v2 dispatcher
	v2Dispatcher := k8sdispatcher.NewK8sModelDispatcher(apiPrefix, app)
	for _, man := range []model.IK8SModelManager{
		k8smodels.NodeManager,
		k8smodels.NamespaceManager,
		k8smodels.LimitRangeManager,
		k8smodels.ResourceQuotaManager,
		k8smodels.PodManager,
		k8smodels.JobManager,
		k8smodels.CronJobManager,
		k8smodels.ServiceManager,
		k8smodels.IngressManager,
		k8smodels.DeploymentManager,
		k8smodels.StatefulSetManager,
		k8smodels.DaemonSetManager,
		k8smodels.SecretManager,
		k8smodels.ConfigMapManager,
		k8smodels.StorageClassManager,
		k8smodels.PVManager,
		k8smodels.PVCManager,
		k8smodels.ClusterRoleManager,
		k8smodels.ClusterRoleBindingManager,
		k8smodels.RoleManager,
		k8smodels.RoleBindingManager,
		k8smodels.ServiceAccountManager,
		k8smodels.EventManager,
	} {
		model.RegisterModelManager(man)
		handler := model.NewK8SModelHandler(man)
		log.Infof("Dispatcher register k8s resource manager %q", man.KeywordPlural())
		v2Dispatcher.Add(handler)
	}

	helmAppPrefix := fmt.Sprintf("%s/releaseapps", apiPrefix)

	for _, man := range []k8s.IK8sResourceManager{
		//meter.MeterAppManager,
		//servicetree.ServicetreeAppManager,
		//notify.NotifyAppManager,
	} {
		handler := k8s.NewK8sResourceHandler(man)
		k8s.AddResourceDispatcher(helmAppPrefix, app, handler)
	}

	k8s.AddHelmDispatcher(apiPrefix, app)
	k8s.AddRawResourceDispatcher(apiPrefix, app)
	k8s.AddMiscDispatcher(apiPrefix, app)
	addDefaultHandler(apiPrefix, app)
}

func addDefaultHandler(apiPrefix string, app *appsrv.Application) {
	app.AddHandler("GET", fmt.Sprintf("%s/version", apiPrefix), appsrv.VersionHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/stats", apiPrefix), appsrv.StatisticHandler)
	app.AddHandler("POST", fmt.Sprintf("%s/ping", apiPrefix), appsrv.PingHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/ping", apiPrefix), appsrv.PingHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/worker_stats", apiPrefix), appsrv.WorkerStatsHandler)
}
