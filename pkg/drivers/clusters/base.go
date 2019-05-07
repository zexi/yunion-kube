package clusters

import (
	"context"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/drivers"
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

func (d *sBaseDriver) NeedGenerateCertificate() bool {
	return false
}

func (d *sBaseDriver) CreateClusterResource(man *clusters.SClusterManager, data *types.CreateClusterData) error {
	// do nothing
	return nil
}

func (d *sBaseDriver) GetAddonsManifest(cluster *clusters.SCluster) (string, error) {
	return "", nil
}

func (d *sBaseDriver) UseClusterAPI() bool {
	return false
}

func (d *sBaseDriver) ValidateCreateMachines(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *clusters.SCluster,
	data []*types.CreateMachineData,
) ([]*types.CreateMachineData, []*types.CreateMachineData, error) {
	var needControlplane bool
	var err error
	var clusterId string
	if cluster != nil {
		clusterId = cluster.GetId()
		needControlplane, err = cluster.NeedControlplane()
	}
	if err != nil {
		return nil, nil, errors.Wrapf(err, "check cluster need controlplane")
	}
	controls, nodes := drivers.GetControlplaneMachineDatas(clusterId, data)
	if needControlplane {
		if len(controls) == 0 {
			return nil, nil, httperrors.NewInputParameterError("controlplane node must created")
		}
	}
	return controls, nodes, nil
}

func (d *sBaseDriver) StartSyncStatus(cluster *clusters.SCluster, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterSyncstatusTask", cluster, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
