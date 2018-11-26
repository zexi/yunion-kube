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

func (m *SCronJobManager) IsRawResource() bool {
	return false
}

func init() {
	CronJobManager = &SCronJobManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("cronjob", "cronjobs"),
	}
}
