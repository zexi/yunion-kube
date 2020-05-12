package models

import (
	"yunion.io/x/onecloud/pkg/appsrv"
)

var (
	syncClusterWorker *appsrv.SWorkerManager
)

func init() {
	syncClusterWorker = appsrv.NewWorkerManager(
		"clusterSyncWorkerManager",
		1,
		2048,
		true,
	)
}

func RunSyncClusterTask(probeF func()) {
	syncClusterWorker.Run(probeF, nil, nil)
}
