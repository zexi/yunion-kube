package tasks

import (
	"context"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/utils/logclient"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/k8s/client/cmd"
)

func init() {
	taskman.RegisterTask(ClusterApplyAddonsTask{})
}

type ClusterApplyAddonsTask struct {
	taskman.STask
}

func (t *ClusterApplyAddonsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	t.SetStage("OnApplyAddons", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		return nil, ApplyAddons(cluster)
	})
}

func (t *ClusterApplyAddonsTask) OnApplyAddons(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterApplyAddonsTask) OnApplyAddonsFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.OnError(ctx, cluster, data)
}

func ApplyAddons(cluster *models.SCluster) error {
	kubeconfig, err := cluster.GetKubeconfig()
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

func (t *ClusterApplyAddonsTask) OnError(ctx context.Context, obj *models.SCluster, err jsonutils.JSONObject) {
	t.SetStageFailed(ctx, err)
	logclient.AddActionLogWithStartable(t, obj, logclient.ActionClusterApplyAddons, err, t.UserCred, false)
}
