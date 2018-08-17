package release

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/helm/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type Release struct {
	*release.Release
}

var emptyList = &ReleaseList{
	ListMeta: dataselect.NewListMeta(),
	Releases: make([]*Release, 0),
}

type ReleaseList struct {
	*dataselect.ListMeta
	Releases []*Release
}

func ToRelease(release *release.Release) *Release {
	return &Release{release}
}

func (r Release) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(r.Release)
}

func (man *SReleaseManager) AllowListItems(req *common.Request) bool {
	return req.AllowListItems()
}

func (man *SReleaseManager) List(req *common.Request) (common.ListResource, error) {
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	return man.GetReleaseList(cli, req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SReleaseManager) GetReleaseList(helmclient *client.HelmTunnelClient, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*ReleaseList, error) {
	list, err := ListReleases(helmclient)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return emptyList, nil
	}
	releaseList, err := ToReleaseList(list.Releases, dsQuery)
	return releaseList, err
}

func (l *ReleaseList) Append(obj interface{}) {
	l.Releases = append(l.Releases, ToRelease(obj.(*release.Release)))
}

func (l *ReleaseList) GetResponseData() interface{} {
	return l.Releases
}

func ToReleaseList(releases []*release.Release, dsQuery *dataselect.DataSelectQuery) (*ReleaseList, error) {
	list := &ReleaseList{
		ListMeta: dataselect.NewListMeta(),
		Releases: make([]*Release, 0),
	}
	err := dataselect.ToResourceList(
		list,
		releases,
		dataselect.NewHelmReleaseDataCell,
		dsQuery,
	)
	return list, err
}

func ListReleases(helmclient *client.HelmTunnelClient) (*rls.ListReleasesResponse, error) {
	stats := []release.Status_Code{
		release.Status_DEPLOYED,
	}
	resp, err := helmclient.ListReleases(
		helm.ReleaseListFilter(""),
		helm.ReleaseListSort(int32(rls.ListSort_LAST_RELEASED)),
		helm.ReleaseListOrder(int32(rls.ListSort_DESC)),
		helm.ReleaseListStatuses(stats),
	)

	if err != nil {
		log.Errorf("Can't retrieve the list of releases: %v", err)
		return nil, err
	}
	return resp, err
}

func ShowRelease(helmclient *client.HelmTunnelClient, releaseName string) (*rls.GetReleaseContentResponse, error) {
	return helmclient.ReleaseContent(releaseName)
}
