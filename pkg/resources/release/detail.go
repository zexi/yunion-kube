package release

import (
	"github.com/pkg/errors"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SReleaseManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetReleaseDetailFromRequest(req, id)
}

func GetReleaseDetailFromRequest(req *common.Request, id string) (*apis.ReleaseDetail, error) {
	namespace := req.GetDefaultNamespace()
	cli, err := req.GetHelmClient(namespace)
	if err != nil {
		return nil, err
	}
	//genericCli, err := req.GetGenericClient()
	//if err != nil {
	//return nil, err
	//}

	detail, err := GetReleaseDetail(cli, req.GetCluster(), req.GetIndexer(), namespace, id)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func GetReleaseDetail(
	helmclient *helm.Client,
	cluster apis.ICluster,
	indexer *client.CacheFactory,
	namespace, releaseName string,
) (*apis.ReleaseDetail, error) {
	log.Infof("Get helm release: %q", releaseName)

	// TODO: find a way to retrieve all the information in a single call

	// 1. We get the information about the release
	rls, err := helmclient.Release().ReleaseContent(releaseName, -1)
	if err != nil {
		return nil, err
	}

	//cfg, err := chartutil.CoalesceValues(rls.Release.Chart, rls.Release.Config)
	//cfg, err := chartutil.ReadValues([]byte(rls.Config.Raw))
	//if err != nil {
	//return nil, fmt.Errorf("CoalesceValues: %v", err)
	//}

	res, err := GetReleaseResources(helmclient, rls, indexer, cluster)
	if err != nil {
		return nil, errors.Wrapf(err, "Get release resources: %v", releaseName)
	}

	return &apis.ReleaseDetail{
		Release:   *ToRelease(rls, cluster),
		Resources: res,
	}, nil
}
