package resourcequota

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

// ResourceQuotaDetailList
type ResourceQuotaDetailList struct {
	apis.ListMeta `json:"listMeta"`
	Items        []api.ResourceQuotaDetail `json:"items"`
}

func ToResourceQuotaDetail(rawResourceQuota *v1.ResourceQuota, cluster api.ICluster) *api.ResourceQuotaDetail {
	statusList := make(map[v1.ResourceName]api.ResourceStatus)

	for key, value := range rawResourceQuota.Status.Hard {
		used := rawResourceQuota.Status.Used[key]
		statusList[key] = api.ResourceStatus{
			Used: used.String(),
			Hard: value.String(),
		}
	}
	return &api.ResourceQuotaDetail{
		ObjectMeta: api.NewObjectMeta(rawResourceQuota.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(rawResourceQuota.TypeMeta),
		Scopes:     rawResourceQuota.Spec.Scopes,
		StatusList: statusList,
	}
}
