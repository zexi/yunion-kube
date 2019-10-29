package limitrange

import (
	"k8s.io/api/core/v1"
	"yunion.io/x/yunion-kube/pkg/apis"
)

// limitRanges provides set of limit ranges by limit types and resource names
type limitRangesMap map[v1.LimitType]rangeMap

// rangeMap provides limit ranges by resource name
type rangeMap map[v1.ResourceName]*apis.LimitRangeItem

func (rMap rangeMap) getRange(resource v1.ResourceName) *apis.LimitRangeItem {
	r, ok := rMap[resource]
	if !ok {
		rMap[resource] = &apis.LimitRangeItem{}
		return rMap[resource]
	} else {
		return r
	}
}

// toLimitRanges converts raw limit ranges to limit ranges map
func toLimitRangesMap(rawLimitRange *v1.LimitRange) limitRangesMap {

	rawLimitRanges := rawLimitRange.Spec.Limits

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

func ToLimitRanges(rawLimitRange *v1.LimitRange) []apis.LimitRangeItem {
	limitRangeMap := toLimitRangesMap(rawLimitRange)
	limitRangeList := make([]apis.LimitRangeItem, 0)
	for limitType, rangeMap := range limitRangeMap {
		for resourceName, limit := range rangeMap {
			limit.ResourceName = resourceName.String()
			limit.ResourceType = string(limitType)
			limitRangeList = append(limitRangeList, *limit)
		}
	}
	return limitRangeList
}
