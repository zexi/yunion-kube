package namespace

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/limitrange"
	rq "yunion.io/x/yunion-kube/pkg/resources/resourcequota"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// NamespaceDetail is a presentation layer view of Kubernetes Namespace resource. This means it is Namespace plus
// additional augmented data we can get from other sources.
type NamespaceDetail struct {
	api.ObjectMeta
	api.TypeMeta

	// Phase is the current lifecycle phase of the namespace.
	Phase v1.NamespacePhase `json:"status"`

	// Events is list of events associated to the namespace.
	EventList *common.EventList `json:"eventList"`

	// ResourceQuotaList is list of resource quotas associated to the namespace
	ResourceQuotaList *rq.ResourceQuotaDetailList `json:"resourceQuotaList"`

	// ResourceLimits is list of limit ranges associated to the namespace
	ResourceLimits []limitrange.LimitRangeItem `json:"resourceLimits"`
}

func (man *SNamespaceManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetNamespaceDetail(req.GetIndexer(), req.GetCluster(), id)
}

// GetNamespaceDetail gets namespace details.
func GetNamespaceDetail(indexer *client.CacheFactory, cluster api.ICluster, name string) (*NamespaceDetail, error) {
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

func toNamespaceDetail(namespace *v1.Namespace, events *common.EventList, resourceQuotaList *rq.ResourceQuotaDetailList, resourceLimits []limitrange.LimitRangeItem, cluster api.ICluster) NamespaceDetail {

	return NamespaceDetail{
		ObjectMeta:        api.NewObjectMeta(namespace.ObjectMeta, cluster),
		TypeMeta:          api.NewTypeMeta(api.ResourceKindNamespace),
		Phase:             namespace.Status.Phase,
		EventList:         events,
		ResourceQuotaList: resourceQuotaList,
		ResourceLimits:    resourceLimits,
	}
}

func getResourceQuotas(indexer *client.CacheFactory, cluster api.ICluster, namespace v1.Namespace) (*rq.ResourceQuotaDetailList, error) {
	list, err := indexer.ResourceQuotaLister().ResourceQuotas(namespace.Name).List(labels.Everything())

	result := &rq.ResourceQuotaDetailList{
		Items:    make([]rq.ResourceQuotaDetail, 0),
		ListMeta: api.ListMeta{Total: len(list)},
	}

	for _, item := range list {
		detail := rq.ToResourceQuotaDetail(item, cluster)
		result.Items = append(result.Items, *detail)
	}

	return result, err
}

func getLimitRanges(indexer *client.CacheFactory, namespace v1.Namespace) ([]limitrange.LimitRangeItem, error) {
	list, err := indexer.LimitRangeLister().LimitRanges(namespace.Name).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	resourceLimits := make([]limitrange.LimitRangeItem, 0)
	for _, item := range list {
		list := limitrange.ToLimitRanges(item)
		resourceLimits = append(resourceLimits, list...)
	}

	return resourceLimits, nil
}
