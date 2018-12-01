package release

import (
	"fmt"
	"k8s.io/helm/pkg/timeconv"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"

	"yunion.io/x/yunion-kube/pkg/helm/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

type ReleaseHistoryInfo struct {
	Revision    int32  `json:"revision"`
	Updated     string `json:"updated"`
	Status      string `json:"status"`
	Chart       string `json:"chart"`
	Description string `json:"description"`
}

func (man *SReleaseManager) AllowGetDetailsHistory(req *common.Request, id string) bool {
	return man.AllowGetItem(req, id)
}

func (man *SReleaseManager) GetDetailsHistory(req *common.Request, id string) (interface{}, error) {
	max, _ := req.Query.Int("max")
	if max == 0 {
		max = 256
	}
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	return GetReleaseHistory(cli, id, int32(max))
}

func GetReleaseHistory(helmclient *client.HelmTunnelClient, name string, max int32) ([]ReleaseHistoryInfo, error) {
	r, err := helmclient.ReleaseHistory(name, helm.WithMaxHistory(max))
	if err != nil {
		return nil, err
	}
	if len(r.Releases) == 0 {
		return nil, nil
	}
	return getReleaseHistory(r.GetReleases()), nil
}

func getReleaseHistory(rls []*release.Release) []ReleaseHistoryInfo {
	ret := make([]ReleaseHistoryInfo, 0)
	for i := len(rls) - 1; i >= 0; i-- {
		r := rls[i]
		c := formatChartname(r.Chart)
		t := timeconv.String(r.Info.LastDeployed)
		s := r.Info.Status.Code.String()
		v := r.Version
		d := r.Info.Description

		rInfo := ReleaseHistoryInfo{
			Revision:    v,
			Updated:     t,
			Status:      s,
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
