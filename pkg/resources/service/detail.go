package service

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/endpoint"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type ServiceDetail struct {
	api.ObjectMeta
	api.TypeMeta

	// InternalEndpoint of all Kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoints is DNS name merged with ports.
	InternalEndpoint common.Endpoint `json:"internalEndpoint"`

	// ExternalEndpoints of all Kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoints is external IP address name merged with ports.
	ExternalEndpoints []common.Endpoint `json:"externalEndpoints"`

	// List of Endpoint obj. that are endpoints of this Service.
	EndpointList endpoint.EndpointList `json:"endpointList"`

	// Label selector of the service.
	Selector map[string]string `json:"selector"`

	// Type determines how the service will be exposed.  Valid options: ClusterIP, NodePort, LoadBalancer
	Type v1.ServiceType `json:"type"`

	// ClusterIP is usually assigned by the master. Valid values are None, empty string (""), or
	// a valid IP address. None can be specified for headless services when proxying is not required
	ClusterIP string `json:"clusterIP"`

	// List of events related to this Service
	EventList []common.Event `json:"events"`

	// PodList represents list of pods targeted by same label selector as this service.
	PodList []pod.Pod `json:"pods"`

	// Show the value of the SessionAffinity of the Service.
	SessionAffinity v1.ServiceAffinity `json:"sessionAffinity"`
}

func (man *SServiceManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetServiceDetail(req.GetIndexer(), req.GetCluster(), namespace, id, req.ToQuery())
}

func GetServiceDetail(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	namespace, name string,
	dsQuery *dataselect.DataSelectQuery,
) (*ServiceDetail, error) {
	log.Infof("Getting details of %s service in %s namespace", name, namespace)
	serviceData, err := indexer.ServiceLister().Services(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	endpointList, err := endpoint.GetServiceEndpoints(indexer, namespace, name)
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
	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannelWithOptions(indexer, common.NewSameNamespaceQuery(namespace), labelSelector),
	}

	apiPodList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	events, err := event.GetPodsEvents(indexer, namespace, apiPodList)
	if err != nil {
		return nil, err
	}

	return pod.ToPodList(apiPodList, events, dsQuery, cluster)
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
func ToServiceDetail(service *v1.Service, events common.EventList, pods pod.PodList, endpointList endpoint.EndpointList, cluster api.ICluster) ServiceDetail {
	return ServiceDetail{
		ObjectMeta:        api.NewObjectMetaV2(service.ObjectMeta, cluster),
		TypeMeta:          api.NewTypeMeta(api.ResourceKindService),
		InternalEndpoint:  common.GetInternalEndpoint(service.Name, service.Namespace, service.Spec.Ports),
		ExternalEndpoints: common.GetExternalEndpoints(service),
		EndpointList:      endpointList,
		Selector:          service.Spec.Selector,
		ClusterIP:         service.Spec.ClusterIP,
		Type:              service.Spec.Type,
		EventList:         events.Events,
		PodList:           pods.Pods,
		SessionAffinity:   service.Spec.SessionAffinity,
	}
}
