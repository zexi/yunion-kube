package tasks

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/request"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

type NodeStartAgentTask struct {
	SNodeBaseTask
}

func init() {
	taskman.RegisterTask(NodeStartAgentTask{})
}

func (t *NodeStartAgentTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.StartKubeAgentOnHost(ctx, obj, data)
}

func (t *NodeStartAgentTask) StartKubeAgentOnHost(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	node := obj.(*models.SNode)
	hostObj, _ := t.Params.Get(models.CLOUD_HOST_DATA_KEY)
	if hostObj == nil {
		t.StartAgentFailed(ctx, node, fmt.Errorf("Not found cloud host info"))
		return
	}
	cloudHost := apis.CloudHost{}
	hostObj.Unmarshal(&cloudHost)
	log.Infof("cloud host: %#v", cloudHost)

	registerConfig, err := node.GetAgentRegisterConfig()
	if err != nil {
		t.StartAgentFailed(ctx, node, err)
		return
	}
	t.SetStage("OnStartAgent", data.(*jsonutils.JSONDict))
	header := http.Header{}
	header.Set("X-Task-Id", t.GetTaskId())
	url := "/kubeagent/start"
	body := jsonutils.Marshal(registerConfig)
	_, err = request.Post(cloudHost.ManagerUrl, t.UserCred.GetTokenString(), url, header, body)
	if err != nil {
		t.StartAgentFailed(ctx, node, err)
	}
}

func (t *NodeStartAgentTask) OnStartAgent(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, data.(*jsonutils.JSONDict))
}

func (t *NodeStartAgentTask) StartAgentFailed(ctx context.Context, obj db.IStandaloneModel, err error) {
	t.OnStartAgentFailed(ctx, obj, jsonutils.NewString(err.Error()))
}

func (t *NodeStartAgentTask) OnStartAgentFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	node := obj.(*models.SNode)
	node.SetStatus(t.UserCred, models.NODE_STATUS_ERROR, data.String())
	t.SetStageFailed(ctx, data.String())
}
