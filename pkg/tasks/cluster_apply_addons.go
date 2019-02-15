package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/k8s/client/cmd"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
)

func init() {
	taskman.RegisterTask(ClusterApplyAddonsTask{})
}

type ClusterApplyAddonsTask struct {
	taskman.STask
}

func (t *ClusterApplyAddonsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	if err := ApplyAddons(cluster); err != nil {
		t.OnError(ctx, cluster, err)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func ApplyAddons(cluster *clusters.SCluster) error {
	kubeconfig, err := cluster.GetKubeConfig()
	if err != nil {
		return err
	}
	cli, err := cmd.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		return err
	}
	manifest, err := cluster.GetDriver().GetAddonsManifest(cluster)
	if err != nil {
		return err
	}
	if len(manifest) == 0 {
		return nil
	}
	return cli.Apply(manifest)
}

func (t *ClusterApplyAddonsTask) OnError(ctx context.Context, machine *clusters.SCluster, err error) {
	t.SetStageFailed(ctx, err.Error())
}
