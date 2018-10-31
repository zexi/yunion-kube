package tasks

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/request"
)

type NodeStopAgentTask struct {
	SNodeBaseTask
}

func init() {
	taskman.RegisterTask(NodeStopAgentTask{})
}

func (t *NodeStopAgentTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.StopKubeAgentOnHost(ctx, obj, data)
}

func (t *NodeStopAgentTask) StopKubeAgentOnHost(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	node := obj.(*models.SNode)
	cloudHost, err := node.GetCloudHost()
	if err != nil {
		t.SetStageFailed(ctx, err.Error())
		return
	}
	t.SetStage("OnStopAgent", data.(*jsonutils.JSONDict))
	header := http.Header{}
	header.Set("X-Task-Id", t.GetTaskId())
	url := "/kubeagent/stop"
	_, err = request.Post(cloudHost.ManagerUrl, t.UserCred.GetTokenString(), url, header, nil)
	if err != nil {
		t.OnFailed(ctx, obj, err)
	}
}

func (t *NodeStopAgentTask) OnStopAgent(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, data.(*jsonutils.JSONDict))
}
