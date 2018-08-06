package taskman

import (
	"fmt"
	"runtime/debug"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/pkg/appsrv"
)

var localTaskWorkerMan *appsrv.WorkerManager

func init() {
	localTaskWorkerMan = appsrv.NewWorkerManager("LocalTaskWorkerManager", 4, 10)
}

func error2TaskData(err error) jsonutils.JSONObject {
	errJson := jsonutils.NewDict()
	errJson.Add(jsonutils.NewString("ERROR"), "__status__")
	errJson.Add(jsonutils.NewString(err.Error()), "reason")
	return errJson
}

func LocalTaskRun(task ITask, proc func() (jsonutils.JSONObject, error)) {
	localTaskWorkerMan.Run(func() {

		log.Debugf("XXXXXXXXXXXXXXXXXXLOCAL TASK RUN STARTXXXXXXXXXXXXXXXXX")
		defer log.Debugf("XXXXXXXXXXXXXXXXXXLOCAL TASK RUN END  XXXXXXXXXXXXXXXXX")

		defer func() {
			if r := recover(); r != nil {
				log.Errorf("LocalTaskRun error: %s", r)
				debug.PrintStack()
				task.ScheduleRun(error2TaskData(fmt.Errorf("LocalTaskRun error: %s", r)))
			}
		}()
		data, err := proc()
		if err != nil {
			task.ScheduleRun(error2TaskData(err))
		} else {
			task.ScheduleRun(data)
		}

	}, nil)
}