package model

import (
	"sort"
	"yunion.io/x/log"

	"k8s.io/apimachinery/pkg/labels"
)

type IQuery interface {
	Namespace(ns string) IQuery
	Limit(limit int64) IQuery
	Offset(offset int64) IQuery
	PagingMarker(marker string) IQuery
	AddFilter(filters ...QueryFilter) IQuery
	FilterAny(any bool) IQuery
	AddOrderFields(ofs ...OrderField) IQuery

	FetchObjects() ([]IK8SModel, error)

	GetTotal() int64
	GetLimit() int64
	GetOffset() int64
}

type QueryFilter func(obj IK8SModel) bool

type sK8SQuery struct {
	limit        int64
	offset       int64
	total        int64
	pagingMarker string
	namespace    string
	filters      []QueryFilter
	filterAny    bool
	orderFields  []OrderField

	cluster ICluster
	manager IK8SModelManager
}

func NewK8SResourceQuery(cluster ICluster, manager IK8SModelManager) IQuery {
	q := &sK8SQuery{
		cluster:     cluster,
		manager:     manager,
		filters:     make([]QueryFilter, 0),
		orderFields: make([]OrderField, 0),
	}
	return q
}

func (q *sK8SQuery) AddFilter(filters ...QueryFilter) IQuery {
	q.filters = append(q.filters, filters...)
	return q
}

func (q *sK8SQuery) FilterAny(any bool) IQuery {
	q.filterAny = any
	return q
}

func (q *sK8SQuery) AddOrderFields(orders ...OrderField) IQuery {
	q.orderFields = append(q.orderFields, orders...)
	return q
}

func (q *sK8SQuery) Namespace(ns string) IQuery {
	q.namespace = ns
	return q
}

func (q *sK8SQuery) Limit(limit int64) IQuery {
	q.limit = limit
	return q
}

func (q sK8SQuery) GetLimit() int64 {
	return q.limit
}

func (q *sK8SQuery) Offset(offset int64) IQuery {
	q.offset = offset
	return q
}

func (q sK8SQuery) GetOffset() int64 {
	return q.offset
}

func (q sK8SQuery) GetTotal() int64 {
	return q.total
}

func (q *sK8SQuery) PagingMarker(pm string) IQuery {
	q.pagingMarker = pm
	return q
}

func (q *sK8SQuery) FetchObjects() ([]IK8SModel, error) {
	cluster := q.cluster
	cli := cluster.GetHandler()
	resInfo := q.manager.GetK8SResourceInfo()
	objs, err := cli.List(resInfo.ResourceName, q.namespace, labels.Everything().String())
	if err != nil {
		return nil, err
	}
	ret := make([]IK8SModel, len(objs))
	for idx, obj := range objs {
		model, err := NewK8SModelObject(q.manager, cluster, obj)
		if err != nil {
			return nil, err
		}
		ret[idx] = model
	}
	ret = q.applyFilters(ret)
	ret = q.applySorters(ret)
	q.total = int64(len(ret))
	ret = q.applyOffseter(ret)
	ret = q.applyLimiter(ret)
	return ret, nil
}

func (q *sK8SQuery) applyFilters(objs []IK8SModel) []IK8SModel {
	// TODO: impl filter any
	ret := make([]IK8SModel, 0)
	for _, obj := range objs {
		filtered := true
		for _, f := range q.filters {
			if !f(obj) {
				filtered = false
				break
			}
		}
		if filtered {
			ret = append(ret, obj)
		}
	}
	return ret
}

var _ sort.Interface = new(k8sModelSorter)

// k8sModelSorter implements sort.Interface
type k8sModelSorter struct {
	objs  []IK8SModel
	field OrderField
}

func newK8SModelSorter(objs []IK8SModel, field OrderField) *k8sModelSorter {
	return &k8sModelSorter{
		objs:  objs,
		field: field,
	}
}

func (s *k8sModelSorter) Len() int {
	return len(s.objs)
}

func (s *k8sModelSorter) Swap(i, j int) {
	s.objs[i], s.objs[j] = s.objs[j], s.objs[i]
}

func (s *k8sModelSorter) Less(i, j int) bool {
	descRet := s.field.Field.Compare(s.objs[i], s.objs[j])
	if s.field.Order == OrderASC {
		return !descRet
	}
	return descRet
}

type K8SModelSorter struct {
	objs   []IK8SModel
	fields []OrderField
}

func (s *K8SModelSorter) doSort() *K8SModelSorter {
	for _, field := range s.fields {
		sorter := newK8SModelSorter(s.objs, field)
		sort.Sort(sorter)
	}
	return s
}

func (s *K8SModelSorter) Objects() []IK8SModel {
	return s.objs
}

func (q *sK8SQuery) applySorters(objs []IK8SModel) []IK8SModel {
	sorter := &K8SModelSorter{
		objs:   objs,
		fields: q.orderFields,
	}
	return sorter.doSort().Objects()
}

func (q *sK8SQuery) applyOffseter(objs []IK8SModel) []IK8SModel {
	ret := objs
	if q.offset == 0 {
		return ret
	}
	if q.total > q.offset {
		ret = ret[q.offset:]
		return ret
	}
	return ret
}

func (q *sK8SQuery) applyLimiter(objs []IK8SModel) []IK8SModel {
	if q.limit < 0 {
		// -1 means not do limit query
		return objs
	}
	log.Errorf("==========q.total: %v, q.limit: %v", q.total, q.limit)
	if q.total > q.limit {
		if q.limit <= int64(len(objs)) {
			return objs[:q.limit]
		}
	}
	return objs
}
