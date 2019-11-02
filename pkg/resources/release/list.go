package release

import (
	//"k8s.io/helm/pkg/helm"
	//"k8s.io/helm/pkg/proto/hapi/release"
	//rls "k8s.io/helm/pkg/proto/hapi/services"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	//"yunion.io/x/yunion-kube/pkg/helm/client"
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type Release struct {
	*release.Release
	// Onecloud cluster data
	*api.ClusterMeta
}

var emptyList = &ReleaseList{
	BaseList: common.NewBaseList(nil),
	Releases: make([]*Release, 0),
}

type ReleaseList struct {
	*common.BaseList
	Releases []*Release
}

func ToRelease(release *release.Release, cluster api.ICluster) *Release {
	return &Release{
		Release:     release,
		ClusterMeta: api.NewClusterMeta(cluster),
	}
}

func (r Release) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(r.Release)
}

func (man *SReleaseManager) List(req *common.Request) (common.ListResource, error) {
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	q := ReleaseListQuery{}
	err = req.Query.Unmarshal(&q)
	if err != nil {
		return nil, err
	}
	q.Namespace = req.GetNamespaceQuery().ToRequestParam()
	return man.GetReleaseList(cli, req.GetCluster(), q, req.ToQuery())
}

type ReleaseListQuery struct {
	Filter     string `json:"filter"`
	All        bool   `json:"all"`
	Namespace  string `json:"namespace"`
	Admin      bool   `json:"admin"`
	Deployed   bool   `json:"deployed"`
	Deleted    bool   `json:"deleted"`
	Deleting   bool   `json:"deleting"`
	Failed     bool   `json:"failed"`
	Superseded bool   `json:"superseded"`
	Pending    bool   `json:"pending"`
}

func (q ReleaseListQuery) statusCodes() []release.Status_Code {
	if q.All {
		return []release.Status_Code{
			release.Status_UNKNOWN,
			release.Status_DEPLOYED,
			release.Status_DELETED,
			release.Status_DELETING,
			release.Status_FAILED,
			release.Status_PENDING_INSTALL,
			release.Status_PENDING_UPGRADE,
			release.Status_PENDING_ROLLBACK,
		}
	}

	status := []release.Status_Code{}
	if q.Deployed {
		status = append(status, release.Status_DEPLOYED)
	}

	if q.Deleted {
		status = append(status, release.Status_DELETED)
	}

	if q.Deleting {
		status = append(status, release.Status_DELETING)
	}

	if q.Failed {
		status = append(status, release.Status_FAILED)
	}

	if q.Superseded {
		status = append(status, release.Status_SUPERSEDED)
	}

	if q.Pending {
		status = append(status, release.Status_PENDING_INSTALL, release.Status_PENDING_UPGRADE, release.Status_PENDING_UPGRADE)
	}

	if len(status) == 0 {
		// Default case
		status = append(status, release.Status_DEPLOYED, release.Status_FAILED, release.Status_PENDING_INSTALL)
	}

	return status
}

func (man *SReleaseManager) GetReleaseList(helmclient *client.HelmTunnelClient, cluster api.ICluster, q ReleaseListQuery, dsQuery *dataselect.DataSelectQuery) (*ReleaseList, error) {
	list, err := ListReleases(helmclient, q)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return emptyList, nil
	}
	releaseList, err := ToReleaseList(list, cluster, dsQuery)
	return releaseList, err
}

func (l *ReleaseList) Append(obj interface{}) {
	l.Releases = append(l.Releases, ToRelease(obj.(*release.Release), l.GetCluster()))
}

func (l *ReleaseList) GetResponseData() interface{} {
	return l.Releases
}

func ToReleaseList(releases []*release.Release, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*ReleaseList, error) {
	list := &ReleaseList{
		BaseList: common.NewBaseList(cluster),
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

func ListReleases(helmclient *client.HelmTunnelClient, q ReleaseListQuery) ([]*release.Release, error) {
	stats := q.statusCodes()
	ops := []helm.ReleaseListOption{
		helm.ReleaseListSort(int32(rls.ListSort_LAST_RELEASED)),
		helm.ReleaseListOrder(int32(rls.ListSort_DESC)),
		helm.ReleaseListStatuses(stats),
	}
	if len(q.Filter) != 0 {
		log.Debugf("Apply filters: %v", q.Filter)
		ops = append(ops, helm.ReleaseListFilter(q.Filter))
	}
	if len(q.Namespace) != 0 {
		ops = append(ops, helm.ReleaseListNamespace(q.Namespace))
	}

	resp, err := helmclient.ListReleases(ops...)
	if err != nil {
		log.Errorf("Can't retrieve the list of releases: %v", err)
		return nil, err
	}
	return filterReleaseList(resp.GetReleases()), err
}

func filterReleaseList(rels []*release.Release) []*release.Release {
	idx := map[string]int32{}

	for _, r := range rels {
		name, version := r.GetName(), r.GetVersion()
		if max, ok := idx[name]; ok {
			// check if we have a greater version already
			if max > version {
				continue
			}
		}
		idx[name] = version
	}

	uniq := make([]*release.Release, 0, len(idx))
	for _, r := range rels {
		if idx[r.GetName()] == r.GetVersion() {
			uniq = append(uniq, r)
		}
	}
	return uniq
}

func ShowRelease(helmclient *client.HelmTunnelClient, releaseName string) (*rls.GetReleaseContentResponse, error) {
	return helmclient.ReleaseContent(releaseName)
}
