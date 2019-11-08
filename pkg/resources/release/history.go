package release

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SReleaseManager) AllowGetDetailsHistory(req *common.Request, id string) bool {
	return man.AllowGetItem(req, id)
}

func (man *SReleaseManager) GetDetailsHistory(req *common.Request, id string) (interface{}, error) {
	max, _ := req.Query.Int("max")
	if max == 0 {
		max = 256
	}
	cli, err := req.GetHelmClient(req.GetDefaultNamespace())
	if err != nil {
		return nil, err
	}
	return GetReleaseHistory(cli.Release(), id, int32(max))
}

func GetReleaseHistory(helmclient helm.IRelease, name string, max int32) ([]apis.ReleaseHistoryInfo, error) {
	r, err := helmclient.History().Run(name)
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	return getReleaseHistory(r), nil
}

func getReleaseHistory(rls []*release.Release) []apis.ReleaseHistoryInfo {
	ret := make([]apis.ReleaseHistoryInfo, 0)
	for i := len(rls) - 1; i >= 0; i-- {
		r := rls[i]
		c := formatChartname(r.Chart)
		t := r.Info.LastDeployed
		s := r.Info.Status
		v := r.Version
		d := r.Info.Description

		rInfo := apis.ReleaseHistoryInfo{
			Revision:    v,
			Updated:     t,
			Status:      string(s),
			Chart:       c,
			Description: d,
		}
		ret = append(ret, rInfo)
	}

	return ret
}

func formatChartname(c *chart.Chart) string {
	if c == nil || c.Metadata == nil {
		// This is an edge case that has happened in prod, though we don't
		// know how: https://github.com/kubernetes/helm/issues/1347
		return "MISSING"
	}
	return fmt.Sprintf("%s-%s", c.Metadata.Name, c.Metadata.Version)
}
