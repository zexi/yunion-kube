package tasks

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	providerv1 "yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

func init() {
	taskman.RegisterTask(MachineCreateTask{})
}

type MachineCreateTask struct {
	taskman.STask
}

func (t *MachineCreateTask) newMachine(machine *machines.SMachine) (*clusterv1.Machine, error) {
	spec := &providerv1.OneCloudMachineProviderSpec{
		ResourceType: machine.ResourceType,
	}
	specVal, err := providerv1.EncodeMachineSpec(spec)
	if err != nil {
		return nil, err
	}
	return &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name: machine.Name,
			Labels: map[string]string{
				"set": string(machine.Role),
			},
		},
		Spec: clusterv1.MachineSpec{
			ProviderSpec: clusterv1.ProviderSpec{
				Value: specVal,
			},
		},
	}, nil
}

func (t *MachineCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*machines.SMachine)
	// TODO: create to cluster api here
	client, err := machine.GetGlobalClient()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	machineObj, err := t.newMachine(machine)
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	_, err = client.ClusterV1alpha1().Machines("").Create(machineObj)
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	log.Infof("Create machines object: %#v", machineObj)
	t.SetStageComplete(ctx, nil)
}

func (t *MachineCreateTask) OnError(ctx context.Context, machine *machines.SMachine, err error) {
	machine.SetStatus(t.UserCred, types.MachineCreateFail, err.Error())
	t.SetStageFailed(ctx, err.Error())
}
