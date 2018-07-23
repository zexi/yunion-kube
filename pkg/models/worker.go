package models

import (
	"yunion.io/yunioncloud/pkg/appsrv"
)

var taskWorkMan *appsrv.WorkerManager

func init() {
	taskWorkMan = appsrv.NewWorkerManager("TaskWorkerManager", 4, 10)
}

func TaskManager() *appsrv.WorkerManager {
	return taskWorkMan
}
