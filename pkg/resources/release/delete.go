package release

import (
	"k8s.io/helm/pkg/helm"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/helm/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SReleaseManager) IsRawResource() bool {
	return false
}

func (man *SReleaseManager) AllowDeleteItem(req *common.Request, id string) bool {
	if man.SNamespaceResourceManager.AllowDeleteItem(req, id) {
		return true
	}

	release, err := GetReleaseDetailFromRequest(req, id)
	if err != nil {
		log.Errorf("Get release detail error: %v", err)
		return false
	}
	namespace := release.Namespace
	return req.UserCred.GetProjectName() == namespace
}

func (man *SReleaseManager) Delete(req *common.Request, id string) error {
	cli, err := req.GetHelmClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	return ReleaseDelete(cli, id)
}

func ReleaseDelete(helmclient *client.HelmTunnelClient, releaseName string) error {
	// TODO: sophisticate command options
	opts := []helm.DeleteOption{
		helm.DeletePurge(true),
	}
	_, err := helmclient.DeleteRelease(releaseName, opts...)
	if err != nil {
		return err
	}
	return nil
}
