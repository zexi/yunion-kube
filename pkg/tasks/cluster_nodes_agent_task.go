package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models"
)

type ClusterNodesAgentTask struct {
	SClusterBaseTask
}

func init() {
	taskman.RegisterTask(ClusterNodesAgentTask{})
}

type nodeAgentTaskFunc func(context.Context, mcclient.TokenCredential, *jsonutils.JSONDict, string) error

func (t *ClusterNodesAgentTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	t.SetStage("OnNodesAgentActionComplete", nil)
	params := t.GetParams()
	action, _ := params.GetString("action")
	for _, obj := range objs {
		var callFunc nodeAgentTaskFunc
		node := obj.(*models.SNode)
		switch action {
		case "start":
			callFunc = node.StartAgentStartTask
		case "restart":
			callFunc = node.StartAgentRestartTask
		case "stop":
			callFunc = node.StartAgentStopTask
		default:
			t.SetStageFailed(ctx, fmt.Sprintf("Invalid node agent task action: %s", action))
			return
		}
		err := callFunc(ctx, t.GetUserCred(), nil, t.GetTaskId())
		cluster, _ := node.GetCluster()
		if err != nil {
			t.SetFailed(ctx, cluster, fmt.Errorf("%s NodeAgent: %v", action, err))
			return
		}
	}
}

func (t *ClusterNodesAgentTask) OnNodesAgentActionComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	t.SetStageComplete(ctx, nil)
}
