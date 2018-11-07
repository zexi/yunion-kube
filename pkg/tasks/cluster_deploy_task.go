package tasks

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/controllers"
	"yunion.io/x/yunion-kube/pkg/models"
)

const (
	CONTAINER_SCHED_TAG = "container"
)

type ClusterDeployTask struct {
	SClusterAgentBaseTask
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
	t.SetStage("OnWaitNodesAgentStart", nil)
	t.OnWaitNodesAgentStart(ctx, obj, data)
}

func (t *ClusterDeployTask) OnWaitNodesAgentStart(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	nodes, err := t.getPendingDeployNodes()
	if err != nil {
		t.SetFailed(ctx, cluster, fmt.Errorf("Get pendingNodes: %v", err))
		return
	}
	if cluster.IsNodeAgentsReady(nodes...) {
		t.SetStage("OnNodeAgentsReady", nil)
		log.Infof("All nodes agent started, start deploy")
		t.doDeploy(ctx, cluster, nodes)
		return
	}
	log.Infof("Not all node ready, wait agents to start")
	err = t.StartNodesAgent(ctx, cluster, nodes, data)
	if err != nil {
		t.SetFailed(ctx, cluster, err)
	}
}

func (t *ClusterDeployTask) OnWaitNodesAgentStartFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj.(*models.SCluster), fmt.Errorf("OnWaitNodesAgentStart: %s", data))
}

func (t *ClusterDeployTask) OnNodeAgentsReady(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Infof("Do nothing when node agents ready")
}

func (t *ClusterDeployTask) doDeploy(ctx context.Context, cluster *models.SCluster, nodes []*models.SNode) {
	postDeployFunc := func() {
		err := addNodesToContainerSchedtag(nodes)
		if err != nil {
			log.Errorf("Add node to container schedtag error: %v", err)
		}
		err = controllers.Manager.AddController(cluster)
		if err != nil {
			log.Errorf("Start controller of cluster %q error: %v", cluster.Name, err)
		}
	}

	err := cluster.Deploy(ctx, postDeployFunc, nodes...)
	if err != nil {
		log.Errorf("Deploy error: %v", err)
		t.SetFailed(ctx, cluster, fmt.Errorf("deploy error: %v", err))
		return
	}

	t.SetStageComplete(ctx, nil)
}

func createContainerSchedtag() error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(CONTAINER_SCHED_TAG), "name")
	params.Add(jsonutils.NewString("Allow run container"), "description")
	s, err := models.GetAdminSession()
	if err != nil {
		return err
	}
	_, err = cloudmod.Schedtags.Create(s, params)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate name") {
			return nil
		}
		return err
	}
	return nil
}

func addNodeToContainerSchedtag(node *models.SNode) error {
	s, err := models.GetAdminSession()
	if err != nil {
		return err
	}
	_, err = cloudmod.Schedtaghosts.Attach(s, CONTAINER_SCHED_TAG, node.Name, nil)
	return err
}

func addNodesToContainerSchedtag(nodes []*models.SNode) error {
	err := createContainerSchedtag()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		err = addNodeToContainerSchedtag(node)
		if err != nil {
			log.Errorf("Add node %s to container schedtag error: %v", node.Name, err)
		}
	}
	return nil
}
