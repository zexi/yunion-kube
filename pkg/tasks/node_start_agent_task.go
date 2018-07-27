package tasks

import (
	"context"
	"fmt"

	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/cloudcommon/db/taskman"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/request"
	"yunion.io/yunion-kube/pkg/types/apis"
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
		t.OnStartAgentFail(ctx, node, fmt.Errorf("Not found cloud host info"))
		return
	}
	cloudHost := apis.CloudHost{}
	hostObj.Unmarshal(&cloudHost)
	log.Infof("cloud host: %#v", cloudHost)

	registerConfig, err := node.GetAgentRegisterConfig()
	if err != nil {
		t.OnStartAgentFail(ctx, node, err)
		return
	}
	url := "/kubeagent/start"
	body := jsonutils.Marshal(registerConfig)
	_, err = request.Post(cloudHost.ManagerUrl, t.UserCred.GetTokenString(), url, nil, body)
	if err != nil {
		t.OnStartAgentFail(ctx, node, err)
	}
	t.OnStartAgentTaskComplete(ctx, node, data)
}

func (t *NodeStartAgentTask) OnStartAgentTaskComplete(ctx context.Context, node *models.SNode, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, data.(*jsonutils.JSONDict))
}

func (t *NodeStartAgentTask) OnStartAgentFail(ctx context.Context, node *models.SNode, err error) {
	errStr := fmt.Sprintf("Start agent on host %q: %v", node.HostId, err)
	node.SetStatus(t.UserCred, models.NODE_STATUS_ERROR, errStr)
	t.SetStageFailed(ctx, errStr)
}
