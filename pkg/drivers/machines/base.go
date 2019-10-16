package machines

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
)

type sBaseDriver struct{}

func newBaseDriver() *sBaseDriver {
	return &sBaseDriver{}
}

func (d *sBaseDriver) ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	man := machines.MachineManager
	name, err := data.GetString("name")
	if err != nil {
		return err
	}
	err = man.ValidateName(name)
	if err != nil {
		return err
	}
	q := man.Query()
	q = man.FilterByName(q, name)
	if q.Count() != 0 {
		return httperrors.NewDuplicateNameError("name", name)
	}
	return nil
}

func (d *sBaseDriver) RequestPrepareMachine(ctx context.Context, userCred mcclient.TokenCredential, machine *machines.SMachine, task taskman.ITask) error {
	/*cluster, err := machine.GetCluster()
	if err != nil {
		return errors.Wrap(err, "GetCluster")
	}*/
	createInput, err := machine.GetCreateInput(userCred)
	if err != nil {
		log.Errorf("Get create input error: %v", err)
	}
	input := &apis.MachinePrepareInput{
		FirstNode: machine.IsFirstNode(),
		Role:      machine.GetRole(),
		Config:    createInput.Config,
	}
	/*input, err = cluster.FillMachinePrepareInput(input)
	if err != nil {
		return errors.Wrap(err, "FillMachinePrepareInput")
	}*/
	return machine.StartPrepareTask(ctx, task.GetUserCred(), jsonutils.Marshal(input).(*jsonutils.JSONDict), task.GetTaskId())
}

func (d *sBaseDriver) PrepareResource(session *mcclient.ClientSession, machine *machines.SMachine, data *apis.MachinePrepareInput) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *sBaseDriver) PostDelete(ctx context.Context, userCred mcclient.TokenCredential, m *machines.SMachine, t taskman.ITask) error {
	t.SetStageComplete(ctx, nil)
	return nil
}

func (d *sBaseDriver) TerminateResource(session *mcclient.ClientSession, machine *machines.SMachine) error {
	return nil
}

func (d *sBaseDriver) GetPrivateIP(session *mcclient.ClientSession, id string) (string, error) {
	return "", fmt.Errorf("not impl")
}

func (d *sBaseDriver) UseClusterAPI() bool {
	return false
}

func (d *sBaseDriver) ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *clusters.SCluster, machine *machines.SMachine) error {
	return nil
}
