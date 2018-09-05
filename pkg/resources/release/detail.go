package release

import (
	"k8s.io/helm/pkg/proto/hapi/release"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/helm/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

type ReleaseDetail struct {
	*release.Release
}

func (man *SReleaseManager) Get(req *common.Request, id string) (interface{}, error) {
	detail, err := GetReleaseDetailFromRequest(req, id)
	if err != nil {
		return nil, err
	}
	return detail.Release, nil
}

func GetReleaseDetailFromRequest(req *common.Request, id string) (*ReleaseDetail, error) {
	namespace := req.GetDefaultNamespace()
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	detail, err := GetReleaseDetail(cli, namespace, id)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func GetReleaseDetail(helmclient *client.HelmTunnelClient, namespace, releaseName string) (*ReleaseDetail, error) {
	log.Infof("Get helm release: %q", releaseName)

	// TODO: find a way to retrieve all the information in a single call

	// 1. We get the information about the release
	rls, err := helmclient.ReleaseContent(releaseName)
	if err != nil {
		return nil, err
	}

	// 2. Now we populate the resources string
	status, err := helmclient.ReleaseStatus(releaseName)
	if err != nil {
		return nil, err
	}
	rls.Release.Info = status.Info
	return &ReleaseDetail{
		Release: rls.Release,
	}, nil
}
