package clusters

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type sBaseDriver struct{}

func newBaseDriver() *sBaseDriver {
	return &sBaseDriver{}
}

func (d *sBaseDriver) ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	return nil
}

func (d *sBaseDriver) ValidateDeleteCondition() error {
	return nil
}

func (d *sBaseDriver) CreateClusterResource(man *clusters.SClusterManager, data *types.CreateClusterData) error {
	// do nothing
	return nil
}

func (d *sBaseDriver) ValidateAddMachine(man *clusters.SClusterManager, machine *types.CreateMachineData) error {
	return nil
}

func (d *sBaseDriver) GetAddonsManifest(cluster *clusters.SCluster) (string, error) {
	return "", nil
}

func (d *sBaseDriver) UseClusterAPI() bool {
	return false
}

func (d *sBaseDriver) RequestDeleteCluster(c *clusters.SCluster) error {
	return fmt.Errorf("Not supported")
}

func (d *sBaseDriver) ValidateAddMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, data []*types.CreateMachineData) error {
	return nil
}

func (d *sBaseDriver) StartSyncStatus(cluster *clusters.SCluster, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterSyncstatusTask", cluster, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
