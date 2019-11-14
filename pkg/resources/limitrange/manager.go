package limitrange

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

var LimitRangeManager *SLimitRanageManager

type SLimitRanageManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	LimitRangeManager = &SLimitRanageManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("limitrange", "limitranges"),
	}
}

func (m *SLimitRanageManager) List(req *common.Request) (common.ListResource, error) {
	query := req.ToQuery()
	return GetLimitRangeList(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), query)
}

func GetLimitRangeList(indexer *client.CacheFactory, cluster api.ICluster, namespace *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*LimitRangeList, error) {
	limits, err := indexer.LimitRangeLister().LimitRanges(namespace.ToRequestParam()).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	return ToLimitRangeList(limits, dsQuery, cluster)
}

type LimitRangeList struct {
	*common.BaseList
	Limits []*api.LimitRange
}

func (l *LimitRangeList) Append(obj interface{}) {
	l.Limits = append(l.Limits, ToLimitRange(obj.(*v1.LimitRange), l.GetCluster()))
}

func (l *LimitRangeList) GetResponseData() interface{} {
	return l.Limits
}

func ToLimitRangeList(limits []*v1.LimitRange, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*LimitRangeList, error) {
	l := &LimitRangeList{
		BaseList: common.NewBaseList(cluster),
		Limits:   make([]*api.LimitRange, 0),
	}
	err := dataselect.ToResourceList(
		l, limits,
		dataselect.NewNamespaceDataCell,
		dsQuery,
	)
	return l, err
}

func ToLimitRange(limit *v1.LimitRange, cluster api.ICluster) *api.LimitRange {
	return &api.LimitRange{
		ObjectMeta:     api.NewObjectMeta(limit.ObjectMeta, cluster),
		TypeMeta:       api.NewTypeMeta(limit.TypeMeta),
		LimitRangeSpec: limit.Spec,
	}
}
