package initial

import (
	"time"

	// "k8s.io/apimachinery/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/manager"

	_ "yunion.io/x/yunion-kube/pkg/drivers/clusters"
	_ "yunion.io/x/yunion-kube/pkg/drivers/machines"
	_ "yunion.io/x/yunion-kube/pkg/tasks"
)

func InitClient(cron *cronman.SCronJobManager) {
	// go wait.Forever(client.BuildApiserverClient, 5*time.Second)
	client.InitClustersManager(manager.ClusterManager())

	cron.AddJobAtIntervalsWithStartRun("StartKubeClusterHealthCheck", time.Minute, models.ClusterManager.ClusterHealthCheckTask, true)
	cron.AddJobAtIntervalsWithStartRun("StartKubeClusterAutoSyncTask", 30*time.Minute, models.ClusterManager.StartAutoSyncTask, true)
}
