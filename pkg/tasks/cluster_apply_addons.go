package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
)

func init() {
	taskman.RegisterTask(ClusterApplyAddonsTask{})
}

type ClusterApplyAddonsTask struct {
	taskman.STask
}

func (t *ClusterApplyAddonsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	machine, err := cluster.GetControlplaneMachine()
	if err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	kubeconfig, err := cluster.GetKubeConfig()
	if err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	if err := machine.(*machines.SMachine).GetDriver().ApplyAddons(cluster, kubeconfig); err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterApplyAddonsTask) OnError(ctx context.Context, machine *clusters.SCluster, err error) {
	t.SetStageFailed(ctx, err.Error())
}
