package models

import (
	"github.com/yunionio/pkg/appsrv"
)

var taskWorkMan *appsrv.WorkerManager

func init() {
	taskWorkMan = appsrv.NewWorkerManager("TaskWorkerManager", 4, 10)
}

func TaskManager() *appsrv.WorkerManager {
	return taskWorkMan
}
