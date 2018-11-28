package service

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

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
	return GetServiceDetail(req.GetK8sClient(), namespace, id, req.ToQuery())
}

func GetServiceDetail(client client.Interface, namespace, name string, dsQuery *dataselect.DataSelectQuery) (*ServiceDetail, error) {
	log.Infof("Getting details of %s service in %s namespace", name, namespace)
	serviceData, err := client.CoreV1().Services(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	endpointList, err := endpoint.GetServiceEndpoints(client, namespace, name)
	if err != nil {
		return nil, err
	}

	podList, err := GetServicePods(client, namespace, name, dsQuery)
	if err != nil {
		return nil, err
	}

	eventList, err := GetServiceEvents(client, dataselect.DefaultDataSelect(), namespace, name)
	if err != nil {
		return nil, err
	}

	service := ToServiceDetail(serviceData, *eventList, *podList, *endpointList)
	return &service, nil
}

// GetServicePods gets list of pods targeted by given label selector in given namespace.
func GetServicePods(cli client.Interface, namespace,
	name string, dsQuery *dataselect.DataSelectQuery) (*pod.PodList, error) {
	service, err := cli.CoreV1().Services(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if service.Spec.Selector == nil {
		return nil, nil
	}

	labelSelector := labels.SelectorFromSet(service.Spec.Selector)
	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannelWithOptions(cli, common.NewSameNamespaceQuery(namespace),
			metaV1.ListOptions{
				LabelSelector: labelSelector.String(),
				FieldSelector: fields.Everything().String(),
			}),
	}

	apiPodList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	events, err := event.GetPodsEvents(cli, namespace, apiPodList.Items)
	if err != nil {
		return nil, err
	}

	return pod.ToPodList(apiPodList.Items, events, dsQuery)
}

// GetServiceEvents returns model events for a service with the given name in the given namespace.
func GetServiceEvents(client client.Interface, dsQuery *dataselect.DataSelectQuery, namespace, name string) (*common.EventList, error) {
	serviceEvents, err := event.GetEvents(client, namespace, name)
	if err != nil {
		return nil, err
	}

	eventList, err := event.CreateEventList(event.FillEventsType(serviceEvents), dsQuery)
	log.Infof("Found %d events related to %s service in %s namespace", len(eventList.Events), name, namespace)
	return &eventList, err
}

// ToServiceDetail returns api service object based on kubernetes service object
func ToServiceDetail(service *v1.Service, events common.EventList, pods pod.PodList, endpointList endpoint.EndpointList) ServiceDetail {
	return ServiceDetail{
		ObjectMeta:        api.NewObjectMeta(service.ObjectMeta),
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
