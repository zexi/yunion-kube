package app

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/k8s"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/resources/cluster"
	"yunion.io/x/yunion-kube/pkg/resources/configmap"
	"yunion.io/x/yunion-kube/pkg/resources/cronjob"
	"yunion.io/x/yunion-kube/pkg/resources/deployment"
	"yunion.io/x/yunion-kube/pkg/resources/ingress"
	"yunion.io/x/yunion-kube/pkg/resources/job"
	"yunion.io/x/yunion-kube/pkg/resources/namespace"
	"yunion.io/x/yunion-kube/pkg/resources/node"
	"yunion.io/x/yunion-kube/pkg/resources/persistentvolume"
	"yunion.io/x/yunion-kube/pkg/resources/persistentvolumeclaim"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	"yunion.io/x/yunion-kube/pkg/resources/rbacroles"
	"yunion.io/x/yunion-kube/pkg/resources/release"
	"yunion.io/x/yunion-kube/pkg/resources/releaseapp/meter"
	"yunion.io/x/yunion-kube/pkg/resources/secret"
	"yunion.io/x/yunion-kube/pkg/resources/service"
	"yunion.io/x/yunion-kube/pkg/resources/statefulset"
	"yunion.io/x/yunion-kube/pkg/resources/storageclass"
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
		models.RepoManager,
	} {
		db.RegisterModelManager(man)
		handler := db.NewModelHandler(man)
		dispatcher.AddModelDispatcher(apiPrefix, app, handler)
	}

	for _, man := range []k8s.IK8sResourceManager{
		configmap.ConfigMapManager,
		cronjob.CronJobManager,
		deployment.DeploymentManager,
		deployment.DeployFromFileManager,
		ingress.IngressManager,
		job.JobManager,
		pod.PodManager,
		namespace.NamespaceManager,
		node.NodeManager,
		persistentvolume.PersistentVolumeManager,
		persistentvolumeclaim.PersistentVolumeClaimManager,
		rbacroles.RbacRoleManager,
		release.ReleaseManager,
		secret.SecretManager,
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
		meter.MeterAppManager,
	} {
		handler := k8s.NewK8sResourceHandler(man)
		k8s.AddResourceDispatcher(helmAppPrefix, app, handler)
	}

	k8s.AddHelmDispatcher(apiPrefix, app)
	k8s.AddRawResourceDispatcher(apiPrefix, app)
	k8s.AddMiscDispatcher(apiPrefix, app)
}
