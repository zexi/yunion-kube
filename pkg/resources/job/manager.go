package job

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var (
	JobManager *SJobManager
)

type SJobManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	JobManager = &SJobManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("job", "jobs"),
	}
}
