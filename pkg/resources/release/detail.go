package release

import (
	"fmt"

	"k8s.io/helm/pkg/chartutil"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/helm/client"
	k8sclient "yunion.io/x/yunion-kube/pkg/k8s/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

type ReleaseDetail struct {
	*Release
	Resources    map[string]interface{} `json:"resources"`
	ConfigValues chartutil.Values       `json:"config_values"`
}

func (man *SReleaseManager) Get(req *common.Request, id string) (interface{}, error) {
	detail, err := GetReleaseDetailFromRequest(req, id)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func GetReleaseDetailFromRequest(req *common.Request, id string) (*ReleaseDetail, error) {
	namespace := req.GetDefaultNamespace()
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	genericCli, err := req.GetGenericClient()
	if err != nil {
		return nil, err
	}

	detail, err := GetReleaseDetail(cli, req.GetCluster(), genericCli, namespace, id)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func GetReleaseDetail(
	helmclient *client.HelmTunnelClient,
	cluster apis.ICluster,
	genericClient *k8sclient.GenericClient,
	namespace, releaseName string,
) (*ReleaseDetail, error) {
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

	//cfg, err := chartutil.CoalesceValues(rls.Release.Chart, rls.Release.Config)
	cfg, err := chartutil.ReadValues([]byte(rls.Release.Config.Raw))
	if err != nil {
		return nil, fmt.Errorf("CoalesceValues: %v", err)
	}

	res, err := GetReleaseResources(genericClient, cluster, rls.Release)
	if err != nil {
		return nil, fmt.Errorf("Get release resources: %v", err)
	}

	return &ReleaseDetail{
		Release:      ToRelease(rls.Release, cluster),
		ConfigValues: cfg,
		Resources:    res,
	}, nil
}
