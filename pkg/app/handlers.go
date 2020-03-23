package app

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/k8s"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	k8sapp "yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/cluster"
	"yunion.io/x/yunion-kube/pkg/resources/configmap"
	"yunion.io/x/yunion-kube/pkg/resources/cronjob"
	//"yunion.io/x/yunion-kube/pkg/resources/daemonset"
	"yunion.io/x/yunion-kube/pkg/resources/deployment"
	"yunion.io/x/yunion-kube/pkg/resources/ingress"
	"yunion.io/x/yunion-kube/pkg/resources/job"
	"yunion.io/x/yunion-kube/pkg/resources/limitrange"
	"yunion.io/x/yunion-kube/pkg/resources/namespace"
	"yunion.io/x/yunion-kube/pkg/resources/node"
	"yunion.io/x/yunion-kube/pkg/resources/persistentvolume"
	"yunion.io/x/yunion-kube/pkg/resources/persistentvolumeclaim"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	//"yunion.io/x/yunion-kube/pkg/resources/pod"
	"yunion.io/x/yunion-kube/pkg/k8smodels"
	"yunion.io/x/yunion-kube/pkg/resources/rbacroles"
	"yunion.io/x/yunion-kube/pkg/resources/release"
	"yunion.io/x/yunion-kube/pkg/resources/resourcequota"
	//"yunion.io/x/yunion-kube/pkg/resources/releaseapp/meter"
	//"yunion.io/x/yunion-kube/pkg/resources/releaseapp/notify"
	//"yunion.io/x/yunion-kube/pkg/resources/releaseapp/servicetree"
	k8sdispatcher "yunion.io/x/yunion-kube/pkg/k8s/dispatcher"
	"yunion.io/x/yunion-kube/pkg/resources/secret"
	"yunion.io/x/yunion-kube/pkg/resources/service"
	"yunion.io/x/yunion-kube/pkg/resources/statefulset"
	"yunion.io/x/yunion-kube/pkg/resources/storageclass"

	_ "yunion.io/x/yunion-kube/pkg/drivers/machines"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	_ "yunion.io/x/yunion-kube/pkg/resources/secret/drivers"
	_ "yunion.io/x/yunion-kube/pkg/resources/storageclass/drivers"
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
		clusters.ClusterManager,
		clusters.X509KeyPairManager,
		clusters.ComponentManager,
		machines.MachineManager,
	} {
		db.RegisterModelManager(man)
		handler := db.NewModelHandler(man)
		dispatcher.AddModelDispatcher(apiPrefix, app, handler)
	}

	for _, man := range []db.IJointModelManager{
		clusters.ClusterX509KeyPairManager,
		clusters.ClusterComponentManager,
	} {
		db.RegisterModelManager(man)
		handler := db.NewJointModelHandler(man)
		dispatcher.AddJointModelDispatcher(apiPrefix, app, handler)
	}

	for _, man := range []k8s.IK8sResourceManager{
		configmap.ConfigMapManager,
		cronjob.CronJobManager,
		k8sapp.AppFromFileManager,
		deployment.DeploymentManager,
		//daemonset.DaemonSetManager,
		ingress.IngressManager,
		job.JobManager,
		pod.PodManager,
		namespace.NamespaceManager,
		limitrange.LimitRangeManager,
		resourcequota.ResourceQuotaManager,
		node.NodeManager,
		persistentvolume.PersistentVolumeManager,
		persistentvolumeclaim.PersistentVolumeClaimManager,
		rbacroles.RbacRoleManager,
		rbacroles.RbacRoleBindingManager,
		rbacroles.ServiceAccountManager,
		release.ReleaseManager,
		secret.SecretManager,
		secret.RegistrySecretManager,
		service.ServiceManager,
		cluster.ClusterManager,
		statefulset.StatefulSetManager,
		storageclass.StorageClassManager,
	} {
		handler := k8s.NewK8sResourceHandler(man)
		k8s.AddResourceDispatcher(apiPrefix, app, handler)
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

	// v2 dispatcher
	v2Dispatcher := k8sdispatcher.NewK8sModelDispatcher(apiPrefix, app)
	for _, man := range []model.IK8SModelManager{
		// k8smodels.PodManager,
		k8smodels.DaemonSetManager,
	} {
		handler := model.NewK8SModelHandler(man)
		v2Dispatcher.Add(handler)
	}
}

func addDefaultHandler(apiPrefix string, app *appsrv.Application) {
	app.AddHandler("GET", fmt.Sprintf("%s/version", apiPrefix), appsrv.VersionHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/stats", apiPrefix), appsrv.StatisticHandler)
	app.AddHandler("POST", fmt.Sprintf("%s/ping", apiPrefix), appsrv.PingHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/ping", apiPrefix), appsrv.PingHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/worker_stats", apiPrefix), appsrv.WorkerStatsHandler)
}
