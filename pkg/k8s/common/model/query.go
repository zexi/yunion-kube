package model

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type IQuery interface {
	Namespace(ns string) IQuery
	Limit(limit int64) IQuery
	Offset(offset int64) IQuery
	PagingMarker(marker string) IQuery
	AddFilter(filters ...QueryFilter) IQuery

	FetchObjects() ([]IK8SModel, error)

	GetTotal() int64
	GetLimit() int64
	GetOffset() int64
}

type QueryFilter func(obj runtime.Object) bool

type sK8SQuery struct {
	limit        int64
	offset       int64
	total        int64
	pagingMarker string
	namespace    string
	filters      []QueryFilter

	cluster ICluster
	manager IK8SModelManager
}

func NewK8SResourceQuery(cluster ICluster, manager IK8SModelManager) IQuery {
	return &sK8SQuery{
		cluster: cluster,
		manager: manager,
		filters: make([]QueryFilter, 0),
	}
}

func (q *sK8SQuery) AddFilter(filters ...QueryFilter) IQuery {
	q.filters = append(q.filters, filters...)
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
	objs = q.applyFilters(objs)
	q.total = int64(len(objs))
	objs = q.applyOffseter(objs)
	objs = q.applyLimiter(objs)
	objs = q.applySorters(objs)
	ret := make([]IK8SModel, len(objs))
	for idx, obj := range objs {
		model, err := NewK8SModelObject(q.manager, cluster, obj)
		if err != nil {
			return nil, err
		}
		ret[idx] = model
	}
	return ret, nil
}

func (q *sK8SQuery) applyFilters(objs []runtime.Object) []runtime.Object {
	ret := make([]runtime.Object, 0)
	for _, obj := range objs {
		filtered := false
		for _, f := range q.filters {
			if f(obj) {
				filtered = true
				continue
			}
		}
		if !filtered {
			ret = append(ret, obj)
		}
	}
	return ret
}

func (q *sK8SQuery) applySorters(objs []runtime.Object) []runtime.Object {
	// TODO
	return objs
}

func (q *sK8SQuery) applyOffseter(objs []runtime.Object) []runtime.Object {
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

func (q *sK8SQuery) applyLimiter(objs []runtime.Object) []runtime.Object {
	if q.limit < 0 {
		// -1 means not do limit query
		return objs
	}
	if q.total > q.limit {
		if q.limit <= int64(len(objs)) {
			return objs[:q.limit]
		}
	}
	return objs
}
