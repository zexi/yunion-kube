package tasks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/request"
)

type NodeRestartAgentTask struct {
	SNodeBaseTask
}

func init() {
	taskman.RegisterTask(NodeRestartAgentTask{})
}

func (t *NodeRestartAgentTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.RestartKubeAgentOnHost(ctx, obj, data)
}

func (t *NodeRestartAgentTask) RestartKubeAgentOnHost(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	node := obj.(*models.SNode)
	cloudHost, err := node.GetCloudHost()
	if err != nil {
		t.OnFailed(ctx, node, err)
		return
	}
	log.Infof("cloud host: %#v", cloudHost)

	t.SetStage("OnRestartAgent", data.(*jsonutils.JSONDict))
	header := http.Header{}
	header.Set("X-Task-Id", t.GetTaskId())
	url := "/kubeagent/restart"
	_, err = request.Post(cloudHost.ManagerUrl, t.UserCred.GetTokenString(), url, header, nil)
	if err != nil {
		t.OnFailed(ctx, node, err)
	}
}

func (t *NodeRestartAgentTask) OnRestartAgent(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	node := obj.(*models.SNode)
	for i := 0; i <= 5; i++ {
		if node.IsAgentReady() {
			log.Infof("[%s] Node %s agent connected", t.GetName(), node.Name)
			t.SetStageComplete(ctx, data.(*jsonutils.JSONDict))
			return
		}
		log.Infof("[%s] Wait node %s agent to connect...", t.GetName(), node.Name)
		time.Sleep(5 * time.Second)
	}
	t.OnFailed(ctx, obj, fmt.Errorf("Node restart agent connection timeout", node.Name))
}

func (t *NodeRestartAgentTask) OnRestartAgentFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.OnFailedJson(ctx, obj.(*models.SNode), data)
}
