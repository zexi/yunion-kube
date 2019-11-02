package release

import (
	//"k8s.io/helm/pkg/helm"
	//"k8s.io/helm/pkg/proto/hapi/release"

	//"yunion.io/x/yunion-kube/pkg/helm/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SReleaseManager) AllowPerformAction(req *common.Request, id string) bool {
	return man.AllowUpdateItem(req, id)
}

func (man *SReleaseManager) AllowPerformRollback(req *common.Request, id string) bool {
	return man.AllowPerformAction(req, id)
}

func (man *SReleaseManager) PerformRollback(req *common.Request, id string) (interface{}, error) {
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	revision, _ := req.Data.Int("revision")
	desc, _ := req.Data.GetString("description")
	return doReleaseRollback(cli, id, int32(revision), desc)
}

func doReleaseRollback(
	cli *client.HelmTunnelClient,
	name string,
	revision int32,
	description string,
) (*release.Release, error) {
	ret, err := cli.RollbackRelease(
		name,
		helm.RollbackRecreate(false),
		helm.RollbackForce(false),
		helm.RollbackDisableHooks(true),
		helm.RollbackVersion(revision),
		helm.RollbackWait(false),
		helm.RollbackDescription(description),
	)
	if err != nil {
		return nil, err
	}
	return ret.GetRelease(), nil
}
