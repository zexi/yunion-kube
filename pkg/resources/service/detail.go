package service

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/endpoint"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
)

func (man *SServiceManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetServiceDetail(req.GetIndexer(), req.GetCluster(), namespace, id, req.ToQuery())
}

func GetServiceDetail(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	namespace, name string,
	dsQuery *dataselect.DataSelectQuery,
) (*api.ServiceDetail, error) {
	log.Infof("Getting details of %s service in %s namespace", name, namespace)
	serviceData, err := indexer.ServiceLister().Services(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	endpointList, err := endpoint.GetServiceEndpoints(indexer, cluster, namespace, name)
	if err != nil {
		return nil, err
	}

	podList, err := GetServicePods(indexer, cluster, namespace, name, dsQuery)
	if err != nil {
		return nil, err
	}

	eventList, err := GetServiceEvents(indexer, cluster, dataselect.DefaultDataSelect(), namespace, name)
	if err != nil {
		return nil, err
	}

	service := ToServiceDetail(serviceData, *eventList, *podList, *endpointList, cluster)
	return &service, nil
}

// GetServicePods gets list of pods targeted by given label selector in given namespace.
func GetServicePods(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	namespace, name string,
	dsQuery *dataselect.DataSelectQuery,
) (*pod.PodList, error) {
	service, err := indexer.ServiceLister().Services(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	if service.Spec.Selector == nil {
		return nil, nil
	}

	labelSelector := labels.SelectorFromSet(service.Spec.Selector)
	nsQuery := common.NewSameNamespaceQuery(namespace)
	channels := &common.ResourceChannels{
		PodList:       common.GetPodListChannelWithOptions(indexer, nsQuery, labelSelector),
		ConfigMapList: common.GetConfigMapListChannel(indexer, nsQuery),
		SecretList:    common.GetSecretListChannel(indexer, nsQuery),
	}

	apiPodList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	return pod.ToPodListByIndexerV2(indexer, apiPodList, namespace, dsQuery, labelSelector, cluster)
}

// GetServiceEvents returns model events for a service with the given name in the given namespace.
func GetServiceEvents(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace, name string) (*common.EventList, error) {
	serviceEvents, err := event.GetEvents(indexer, namespace, name)
	if err != nil {
		return nil, err
	}

	eventList, err := event.CreateEventList(event.FillEventsType(serviceEvents), dsQuery, cluster)
	log.Infof("Found %d events related to %s service in %s namespace", len(eventList.Events), name, namespace)
	return eventList, err
}

// ToServiceDetail returns api service object based on kubernetes service object
func ToServiceDetail(service *v1.Service, events common.EventList, pods pod.PodList, endpointList endpoint.EndpointList, cluster api.ICluster) api.ServiceDetail {
	return api.ServiceDetail{
		Service:         ToService(service, cluster),
		EndpointList:    endpointList.Endpoints,
		ClusterIP:       service.Spec.ClusterIP,
		Type:            service.Spec.Type,
		EventList:       events.Events,
		PodList:         pods.Pods,
		SessionAffinity: service.Spec.SessionAffinity,
	}
}
