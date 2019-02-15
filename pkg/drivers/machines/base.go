package machines

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
)

type sBaseDriver struct{}

func newBaseDriver() *sBaseDriver {
	return &sBaseDriver{}
}

func (d *sBaseDriver) ValidateCreateData(session *mcclient.ClientSession, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
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

func (d *sBaseDriver) PrepareResource(session *mcclient.ClientSession, machine *machines.SMachine, data *machines.MachinePrepareData) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *sBaseDriver) TerminateResource(session *mcclient.ClientSession, machine *machines.SMachine) error {
	return nil
}

func (d *sBaseDriver) GetPrivateIP(session *mcclient.ClientSession, id string) (string, error) {
	return "", fmt.Errorf("not impl")
}

// TODO: mv to cluster driver
func (d *sBaseDriver) ApplyAddons(cluster *clusters.SCluster, kubeconfig string) error {
	return fmt.Errorf("not impl")
}
