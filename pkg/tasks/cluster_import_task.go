package tasks

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	ykecluster "yunion.io/yke/pkg/cluster"

	"yunion.io/x/yunion-kube/pkg/models"
)

type ClusterImportTask struct {
	SClusterBaseTask
}

func init() {
	taskman.RegisterTask(ClusterImportTask{})
}

func (t *ClusterImportTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	cluster.SetStatus(t.UserCred, models.CLUSTER_STATUS_IMPORT, "")
	nodes, err := cluster.GetNodes()
	if err != nil {
		t.SetFailed(ctx, cluster, err)
		return
	}
	err = t.removeCNINetworkPlugin(ctx, cluster)
	if err != nil {
		t.SetFailed(ctx, cluster, err)
		return
	}
	t.StartDeployCluster(ctx, cluster, nodes)
}

func (t *ClusterImportTask) removeCNINetworkPlugin(ctx context.Context, cluster *models.SCluster) error {
	k8sCli, err := cluster.GetK8sClient()
	if err != nil {
		return err
	}
	jobName := fmt.Sprintf("%s-deploy-job", ykecluster.NetworkPluginResourceName)
	err = k8sCli.BatchV1().Jobs("kube-system").Delete(jobName, &metav1.DeleteOptions{})
	if err != nil {
		log.Errorf("Failed to delete job: %q", jobName)
	}
	err = k8sCli.AppsV1beta2().DaemonSets("kube-system").Delete("yunion", &metav1.DeleteOptions{})
	if err != nil {
		log.Errorf("Failed to delete DaemonSets: yunion")
	}
	return nil
}

func (t *ClusterImportTask) StartDeployCluster(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode) {
	t.SetStage("OnDeployComplete", nil)
	cluster.StartClusterDeployTask(ctx, t.UserCred, models.FetchClusterDeployTaskData(nodes), t.GetTaskId())
}

func (t *ClusterImportTask) OnDeployComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterImportTask) OnDeployCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj.(*models.SCluster), fmt.Errorf("OnDeployCompleteFailed: %s", data))
}
