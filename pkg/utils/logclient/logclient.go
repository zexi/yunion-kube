package logclient

import "yunion.io/x/onecloud/pkg/util/logclient"

const (
	ActionClusterCreate         = "创建集群"
	ActionClusterCreateMachines = "创建机器"
	ActionClusterAddMachine     = "添加机器"
	ActionClusterDeleteMachine  = "删除机器"
	ActionClusterDelete         = "删除集群"
	ActionClusterApplyAddons    = "部署插件"
	ActionClusterSyncStatus     = "同步状态"
	ActionClusterSync           = "同步"

	ActionMachineCreate  = "创建机器"
	ActionMachinePrepare = "准备机器"
	ActionMachineDelete  = "删除机器"
)

var (
	AddActionLogWithStartable = logclient.AddActionLogWithStartable
)
