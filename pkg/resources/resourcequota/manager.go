package resourcequota

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

var ResourceQuotaManager *SResourceQuotaManager

type SResourceQuotaManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	ResourceQuotaManager = &SResourceQuotaManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("resourcequota", "resourcequotas"),
	}
}

func (m *SResourceQuotaManager) List(req *common.Request) (common.ListResource, error) {
	query := req.ToQuery()
	return GetResourceQuotaList(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), query)
}

func GetResourceQuotaList(indexer *client.CacheFactory, cluster api.ICluster, namespace *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	rs, err := indexer.ResourceQuotaLister().ResourceQuotas(namespace.ToRequestParam()).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	return ToResourceQuotaList(rs, dsQuery, cluster)
}

type ResourceQuotaList struct {
	*common.BaseList
	ResourceQuotas []*api.ResourceQuota
}

func (l *ResourceQuotaList) Append(obj interface{}) {
	l.ResourceQuotas = append(l.ResourceQuotas, ToResourceQuota(obj.(*v1.ResourceQuota), l.GetCluster()))
}

func (l *ResourceQuotaList) GetResponseData() interface{} {
	return l.ResourceQuotas
}

func ToResourceQuotaList(rs []*v1.ResourceQuota, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*ResourceQuotaList, error) {
	l := &ResourceQuotaList{
		BaseList:       common.NewBaseList(cluster),
		ResourceQuotas: make([]*api.ResourceQuota, 0),
	}
	err := dataselect.ToResourceList(l, rs,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return l, err
}

func ToResourceQuota(rs *v1.ResourceQuota, cluster api.ICluster) *api.ResourceQuota {
	return &api.ResourceQuota{
		ObjectMeta:        api.NewObjectMeta(rs.ObjectMeta, cluster),
		TypeMeta:          api.NewTypeMeta(rs.TypeMeta),
		ResourceQuotaSpec: rs.Spec,
	}
}
