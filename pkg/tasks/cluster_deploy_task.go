package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models"
)

type ClusterDeployTask struct {
	SClusterBaseTask
}

func init() {
	taskman.RegisterTask(ClusterDeployTask{})
}

func (t *ClusterDeployTask) getPendingDeployNodes() ([]*models.SNode, error) {
	nodeIds, err := t.GetParams().GetArray(models.NODES_DEPLOY_IDS_KEY)
	if err != nil {
		return nil, err
	}
	ret := make([]*models.SNode, len(nodeIds))
	for i, idObj := range nodeIds {
		id, _ := idObj.GetString()
		node, err := models.NodeManager.FetchNodeById(id)
		if err != nil {
			return nil, err
		}
		ret[i] = node
	}
	return ret, nil
}

func (t *ClusterDeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	nodes, err := t.getPendingDeployNodes()
	if err != nil {
		t.SetFailed(ctx, cluster, err)
		return
	}
	cluster.Deploy(ctx, nodes...)
	t.SetStageComplete(ctx, nil)
}
