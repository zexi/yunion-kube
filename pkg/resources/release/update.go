package release

import (
	"strings"

	"helm.sh/helm/v3/pkg/release"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SReleaseManager) AllowUpdateItem(req *common.Request, id string) bool {
	return man.AllowDeleteItem(req, id)
}

func (man *SReleaseManager) Update(req *common.Request, id string) (interface{}, error) {
	input := &api.ReleaseUpdateInput{}
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	cli, err := req.GetHelmClient(input.Namespace)
	if err != nil {
		return nil, err
	}
	return ReleaseUpgrade(cli.Release(), input)
}

func ReleaseUpgrade(helmclient helm.IRelease, opt *api.ReleaseUpdateInput) (*release.Release, error) {
	log.Infof("Upgrade chart=%q, release name=%q", opt.ChartName, opt.ReleaseName)
	segs := strings.Split(opt.ChartName, "/")
	if len(segs) != 2 {
		return nil, httperrors.NewInputParameterError("Illegal chart name: %q", opt.ChartName)
	}
	return helmclient.Update(opt)
}
