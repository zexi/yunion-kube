package release

import (
	"fmt"
	"strings"

	//"k8s.io/helm/pkg/helm"
	//rls "k8s.io/helm/pkg/proto/hapi/services"

	"yunion.io/x/log"

	//"yunion.io/x/yunion-kube/pkg/helm/client"
	//helmdata "yunion.io/x/yunion-kube/pkg/helm/data"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SReleaseManager) AllowUpdateItem(req *common.Request, id string) bool {
	return man.AllowDeleteItem(req, id)
}

func (man *SReleaseManager) Update(req *common.Request, id string) (interface{}, error) {
	updateOpt, err := NewCreateUpdateReleaseReq(req.Data)
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	ret, err := ReleaseUpgrade(cli, updateOpt)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func ReleaseUpgrade(helmclient *client.HelmTunnelClient, opt *CreateUpdateReleaseRequest) (*rls.UpdateReleaseResponse, error) {
	log.Infof("Upgrade chart=%q, release name=%q", opt.ChartName, opt.ReleaseName)
	segs := strings.Split(opt.ChartName, "/")
	if len(segs) != 2 {
		return nil, fmt.Errorf("Illegal chart name: %q", opt.ChartName)
	}
	repoName, chartName := segs[0], segs[1]
	pkg, err := helmdata.ChartFromRepo(repoName, chartName, opt.Version)
	if err != nil {
		return nil, err
	}
	chartRequest := pkg.Chart
	rawVals, err := opt.Vals()
	if err != nil {
		return nil, err
	}
	upgradeRes, err := helmclient.UpdateReleaseFromChart(
		opt.ReleaseName,
		chartRequest,
		helm.UpdateValueOverrides(rawVals),
		helm.UpgradeDisableHooks(true),
		helm.UpgradeDryRun(opt.DryRun),
		helm.UpgradeTimeout(opt.Timeout),
		helm.ResetValues(opt.ResetValues),
		helm.ReuseValues(opt.ReUseValues),
	)
	if err != nil {
		return nil, fmt.Errorf("Error upgrade chart: %v", err)
	}
	return upgradeRes, nil
}
