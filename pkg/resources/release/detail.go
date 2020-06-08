package release

import (
	"github.com/pkg/errors"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	"yunion.io/x/yunion-kube/pkg/resources/chart"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SReleaseManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetReleaseDetailFromRequest(req, id)
}

func GetReleaseDetailFromRequest(req *common.Request, id string) (*api.ReleaseDetail, error) {
	namespace := req.GetDefaultNamespace()
	cli, err := req.GetHelmClient(namespace)
	if err != nil {
		return nil, err
	}

	detail, err := GetReleaseDetail(cli, req.GetCluster(), req.ClusterManager, req.GetIndexer(), namespace, id)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func GetReleaseDetail(
	helmclient *helm.Client,
	cluster api.ICluster,
	clusterMan model.ICluster,
	indexer *client.CacheFactory,
	namespace, releaseName string,
) (*api.ReleaseDetail, error) {
	log.Infof("Get helm release: %q", releaseName)

	rls, err := helmclient.Release().ReleaseContent(releaseName, -1)
	if err != nil {
		return nil, err
	}

	res, err := GetReleaseResources(helmclient, rls, indexer, cluster, clusterMan)
	if err != nil {
		return nil, errors.Wrapf(err, "Get release resources: %v", releaseName)
	}

	return &api.ReleaseDetail{
		Release:   *ToRelease(rls, cluster),
		Resources: res,
		Files:     chart.GetChartRawFiles(rls.Chart),
	}, nil
}
