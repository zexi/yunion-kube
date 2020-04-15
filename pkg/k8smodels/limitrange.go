package k8smodels

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	LimitRangeManager *SLimitRangeManager
	_                 model.IK8SModel = &SLimitRange{}
)

func init() {
	LimitRangeManager = &SLimitRangeManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			new(SLimitRange),
			"limitrange",
			"limitranges"),
	}
	LimitRangeManager.SetVirtualObject(LimitRangeManager)
}

type SLimitRangeManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SLimitRange struct {
	model.SK8SClusterResourceBase
}

func (m *SLimitRangeManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameLimitRange,
		Object:       &v1.LimitRange{},
		KindName:     apis.KindNameLimitRange,
	}
}

func (m *SLimitRangeManager) GetRawLimitRanges(cluster model.ICluster, ns string) ([]*v1.LimitRange, error) {
	return cluster.GetHandler().GetIndexer().
		LimitRangeLister().LimitRanges(ns).List(labels.Everything())
}

func (m *SLimitRangeManager) GetLimitRanges(cluster model.ICluster, ns string) ([]*apis.LimitRange, error) {
	lrs, err := m.GetRawLimitRanges(cluster, ns)
	if err != nil {
		return nil, err
	}
	ret := make([]*apis.LimitRange, 0)
	for idx := range lrs {
		item, err := m.GetLimitRange(cluster, lrs[idx])
		if err != nil {
			return nil, err
		}
		ret = append(ret, item)
	}
	return ret, nil
}

func (m *SLimitRangeManager) GetLimitRange(cluster model.ICluster, obj *v1.LimitRange) (*apis.LimitRange, error) {
	mObj, err := model.NewK8SModelObject(m, cluster, obj)
	if err != nil {
		return nil, err
	}
	return mObj.(*SLimitRange).GetAPIObject()
}

func (obj *SLimitRange) GetRawLimitRange() *v1.LimitRange {
	return obj.GetK8SObject().(*v1.LimitRange)
}

func (obj *SLimitRange) GetAPIObject() (*apis.LimitRange, error) {
	return &apis.LimitRange{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Limits:     obj.ToRangeItem(),
	}, nil
}

func (obj *SLimitRange) GetAPIDetailObject() (*apis.LimitRangeDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	return &apis.LimitRangeDetail{
		LimitRange: *apiObj,
	}, nil
}

func (m *SLimitRangeManager) ToRangeItems(cluster model.ICluster, lrs []*v1.LimitRange) ([]*apis.LimitRangeItem, error) {
	ret := make([]*apis.LimitRangeItem, 0)
	for _, item := range lrs {
		mObj, err := model.NewK8SModelObject(m, cluster, item)
		if err != nil {
			return nil, err
		}
		list := mObj.(*SLimitRange).ToRangeItem()
		ret = append(ret, list...)
	}
	return ret, nil
}

func (obj *SLimitRange) ToRangeItem() []*apis.LimitRangeItem {
	lr := obj.GetRawLimitRange()
	limitRangeMap := obj.toLimitRangesMap(lr)
	limitRangeList := make([]*apis.LimitRangeItem, 0)
	for limitType, rangeMap := range limitRangeMap {
		for resourceName, limit := range rangeMap {
			limit.ResourceName = resourceName.String()
			limit.ResourceType = string(limitType)
			limitRangeList = append(limitRangeList, limit)
		}
	}
	return limitRangeList
}

type limitRangesMap map[v1.LimitType]rangeMap

type rangeMap map[v1.ResourceName]*apis.LimitRangeItem

func (rMap rangeMap) getRange(resource v1.ResourceName) *apis.LimitRangeItem {
	r, ok := rMap[resource]
	if !ok {
		rMap[resource] = &apis.LimitRangeItem{}
		return rMap[resource]
	}
	return r
}

func (obj *SLimitRange) toLimitRangesMap(lr *v1.LimitRange) limitRangesMap {
	rawLimitRanges := lr.Spec.Limits

	limitRanges := make(limitRangesMap, len(rawLimitRanges))

	for _, rawLimitRange := range rawLimitRanges {

		rangeMap := make(rangeMap)

		for resource, min := range rawLimitRange.Min {
			rangeMap.getRange(resource).Min = min.String()
		}

		for resource, max := range rawLimitRange.Max {
			rangeMap.getRange(resource).Max = max.String()
		}

		for resource, df := range rawLimitRange.Default {
			rangeMap.getRange(resource).Default = df.String()
		}

		for resource, dfR := range rawLimitRange.DefaultRequest {
			rangeMap.getRange(resource).DefaultRequest = dfR.String()
		}

		for resource, mLR := range rawLimitRange.MaxLimitRequestRatio {
			rangeMap.getRange(resource).MaxLimitRequestRatio = mLR.String()
		}

		limitRanges[rawLimitRange.Type] = rangeMap
	}

	return limitRanges
}
