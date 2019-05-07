package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type ClusterCreateMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterCreateMachinesTask{})
}

func (t *ClusterCreateMachinesTask) getMachines(cluster *clusters.SCluster) ([]*types.CreateMachineData, error) {
	params := t.GetParams()
	ret := []*types.CreateMachineData{}
	ms := []types.CreateMachineData{}
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

func (t *ClusterCreateMachinesTask) createMachines(ctx context.Context, cluster *clusters.SCluster, ms []*types.CreateMachineData) error {
	return cluster.CreateMachines(ctx, t.GetUserCred(), ms, t)
}

func (t *ClusterCreateMachinesTask) OnMachinesCreated(ctx context.Context, cluster *clusters.SCluster, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterCreateMachinesTask) OnMachinesCreatedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.onError(ctx, obj.(*clusters.SCluster), fmt.Errorf(data.String()))
}

func (t *ClusterCreateMachinesTask) onError(ctx context.Context, cluster *clusters.SCluster, err error) {
	SetObjectTaskFailed(ctx, t, cluster, types.ClusterStatusCreateMachineFail, err.Error())
}
