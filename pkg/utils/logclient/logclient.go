package logclient

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

const (
	ActionClusterCreate         TEventAction = "cluster_create"
	ActionClusterCreateMachines TEventAction = "cluster_create_machines"
	ActionClusterAddMachine     TEventAction = "cluster_add_machine"
	ActionClusterDeleteMachine  TEventAction = "cluster_delete_machine"
	ActionClusterDelete         TEventAction = "cluster_delete"
	ActionClusterApplyAddons    TEventAction = "cluster_apply_addons"
	ActionClusterSyncStatus     TEventAction = "cluster_sync_status"
	ActionClusterSync           TEventAction = "cluster_sync"

	ActionMachineCreate  TEventAction = "machine_create"
	ActionMachinePrepare TEventAction = "machine_prepare"
	ActionMachineDelete  TEventAction = "machine_delete"

	ActionResourceCreate TEventAction = "resource_create"
	ActionResourceUpdate TEventAction = "resource_update"
	ActionResourceDelete TEventAction = "resource_delete"
	ActionResourceSync   TEventAction = "resource_sync"
)

var (
	EventActionMap map[TEventAction]string
)

type TEventAction string

func init() {
	EventActionMap = map[TEventAction]string{
		ActionClusterCreate:         "创建集群",
		ActionClusterCreateMachines: "创建机器",
		ActionClusterAddMachine:     "添加机器",
		ActionClusterDeleteMachine:  "删除机器",
		ActionClusterDelete:         "删除集群",
		ActionClusterApplyAddons:    "部署插件",
		ActionClusterSyncStatus:     "同步状态",
		ActionClusterSync:           "同步",
		ActionMachineCreate:         "创建机器",
		ActionMachinePrepare:        "准备机器",
		ActionMachineDelete:         "删除机器",
		ActionResourceCreate:        "创建资源",
		ActionResourceUpdate:        "更新资源",
		ActionResourceDelete:        "删除资源",
	}
}

// LogWithStartable log record with start time
func LogWithStartable(task cloudcommon.IStartable, obj db.IModel, eventAction TEventAction, iNotes interface{}, userCred mcclient.TokenCredential, isSuccess bool) {
	logAction, ok := EventActionMap[eventAction]
	if !ok {
		logAction = string(eventAction)
	}
	db.OpsLog.LogEvent(obj, string(eventAction), iNotes, userCred)
	logclient.AddActionLogWithStartable(task, obj, logAction, iNotes, userCred, isSuccess)
}
