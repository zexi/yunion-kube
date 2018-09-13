package cronjob

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var (
	CronJobManager *SCronJobManager
)

type SCronJobManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	CronJobManager = &SCronJobManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("cronjob", "cronjobs"),
	}
}
