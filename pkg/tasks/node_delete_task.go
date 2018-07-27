package tasks

import (
	"context"
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/yunion-kube/pkg/models"
)

type NodeDeleteTask struct {
	SNodeBaseTask
}

func init() {
	taskman.RegisterTask(NodeDeleteTask{})
}

func (t *NodeDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	node := obj.(*models.SNode)
	err := node.CleanUpComponents(ctx, data)
	if err != nil {
		t.SetStageFailed(ctx, node, fmt.Sprintf("CleanUp components: %v", err))
		return
	}
	err = t.StartNodeDelete(ctx, node)
	if err != nil {
		t.SetStageFailed(ctx, node, fmt.Sprintf("Do node delete: %v", err))
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *NodeDeleteTask) StartNodeDelete(ctx context.Context, node *models.SNode) error {
	err := node.RemoveNodeFromCluster(ctx)
	if err != nil {
		return err
	}
	// TODO: stop agent on host
	err = node.RealDelete(ctx, t.UserCred)
	if err != nil {
		return err
	}
	return nil
}

func (t *NodeDeleteTask) SetStageFailed(ctx context.Context, node *models.SNode, reason string) {
	node.SetStatus(t.UserCred, models.NODE_STATUS_ERROR, reason)
	t.STask.SetStageFailed(ctx, reason)
}
