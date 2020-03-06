package namespace

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/limitrange"
	rq "yunion.io/x/yunion-kube/pkg/resources/resourcequota"
)

func (man *SNamespaceManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetNamespaceDetail(req.GetIndexer(), req.GetCluster(), id)
}

// GetNamespaceDetail gets namespace details.
func GetNamespaceDetail(indexer *client.CacheFactory, cluster api.ICluster, name string) (*api.NamespaceDetail, error) {
	log.Infof("Getting details of %s namespace", name)

	namespace, err := indexer.NamespaceLister().Get(name)
	if err != nil {
		return nil, err
	}

	events, err := event.GetNamespaceEvents(indexer, cluster, dataselect.DefaultDataSelect(), namespace.Name)
	if err != nil {
		return nil, err
	}

	resourceQuotaList, err := getResourceQuotas(indexer, cluster, *namespace)
	if err != nil {
		return nil, err
	}

	resourceLimits, err := getLimitRanges(indexer, *namespace)
	if err != nil {
		return nil, err
	}

	namespaceDetails := toNamespaceDetail(namespace, events, resourceQuotaList, resourceLimits, cluster)
	return &namespaceDetails, nil
}

func toNamespaceDetail(namespace *v1.Namespace, events *common.EventList, resourceQuotaList *rq.ResourceQuotaDetailList, resourceLimits []api.LimitRangeItem, cluster api.ICluster) api.NamespaceDetail {

	return api.NamespaceDetail{
		Namespace:         toNamespace(namespace, cluster),
		EventList:         events.Events,
		ResourceQuotaList: resourceQuotaList.Items,
		ResourceLimits:    resourceLimits,
	}
}

func getResourceQuotas(indexer *client.CacheFactory, cluster api.ICluster, namespace v1.Namespace) (*rq.ResourceQuotaDetailList, error) {
	list, err := indexer.ResourceQuotaLister().ResourceQuotas(namespace.Name).List(labels.Everything())

	result := &rq.ResourceQuotaDetailList{
		Items: make([]api.ResourceQuotaDetail, 0),
	}

	for _, item := range list {
		detail := rq.ToResourceQuotaDetail(item, cluster)
		result.Items = append(result.Items, *detail)
	}

	return result, err
}

func getLimitRanges(indexer *client.CacheFactory, namespace v1.Namespace) ([]api.LimitRangeItem, error) {
	list, err := indexer.LimitRangeLister().LimitRanges(namespace.Name).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	resourceLimits := make([]api.LimitRangeItem, 0)
	for _, item := range list {
		list := limitrange.ToLimitRanges(item)
		resourceLimits = append(resourceLimits, list...)
	}

	return resourceLimits, nil
}
