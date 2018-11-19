package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func RunBatchTask(
	ctx context.Context,
	items []db.IStandaloneModel,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
	taskName, parentTaskId string,
) {
	params := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewParallelTask(ctx, taskName, items, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s newTask error %s", taskName, err)
		return
	}
	task.ScheduleRun(nil)
}

func getNodesById(nodes []*SNode, id string) *SNode {
	for _, node := range nodes {
		if node.Id == id || node.Name == id {
			return node
		}
	}
	return nil
}
