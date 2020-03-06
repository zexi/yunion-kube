package tasks

import (
	"context"
	"fmt"
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/utils/logclient"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
)

type ClusterCreateMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterCreateMachinesTask{})
}

func (t *ClusterCreateMachinesTask) getMachines(cluster *clusters.SCluster) ([]*apis.CreateMachineData, error) {
	params := t.GetParams()
	ret := []*apis.CreateMachineData{}
	ms := []apis.CreateMachineData{}
	if err := params.Unmarshal(&ms, "machines"); err != nil {
		return nil, err
	}
	for _, m := range ms {
		m.ClusterId = cluster.Id
		tmp := m
		ret = append(ret, &tmp)
	}
	return ret, nil
}

func (t *ClusterCreateMachinesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*clusters.SCluster)
	machines, err := t.getMachines(cluster)
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	t.SetStage("OnMachinesCreated", nil)
	if err := t.createMachines(ctx, cluster, machines); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
}

func (t *ClusterCreateMachinesTask) createMachines(ctx context.Context, cluster *clusters.SCluster, ms []*apis.CreateMachineData) error {
	return cluster.CreateMachines(ctx, t.GetUserCred(), ms, t)
}

func (t *ClusterCreateMachinesTask) OnMachinesCreated(ctx context.Context, cluster *clusters.SCluster, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(t, cluster, logclient.ActionClusterCreateMachines, nil, t.UserCred, true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterCreateMachinesTask) OnMachinesCreatedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.onError(ctx, obj.(*clusters.SCluster), fmt.Errorf(data.String()))
}

func (t *ClusterCreateMachinesTask) onError(ctx context.Context, cluster *clusters.SCluster, err error) {
	SetObjectTaskFailed(ctx, t, cluster, apis.ClusterStatusCreateMachineFail, err.Error())
	logclient.AddActionLogWithStartable(t, cluster, logclient.ActionClusterCreateMachines, err.Error(), t.UserCred, false)
}
