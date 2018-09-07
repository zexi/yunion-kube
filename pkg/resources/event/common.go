package event

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// GetEvents gets events associated to resource with given name.
func GetEvents(client client.Interface, namespace, resourceName string) ([]v1.Event, error) {
	fieldSelector, err := fields.ParseSelector("involvedObject.name" + "=" + resourceName)

	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		EventList: common.GetEventListChannelWithOptions(
			client,
			common.NewSameNamespaceQuery(namespace),
			metaV1.ListOptions{
				LabelSelector: labels.Everything().String(),
				FieldSelector: fieldSelector.String(),
			}),
	}

	eventList := <-channels.EventList.List
	if err := <-channels.EventList.Error; err != nil {
		return nil, err
	}

	return FillEventsType(eventList.Items), nil
}

// GetNamespaceEvents gets events associated to a namespace with given name.
func GetNamespaceEvents(client client.Interface, dsQuery *dataselect.DataSelectQuery, namespace string) (common.EventList, error) {
	events, _ := client.CoreV1().Events(namespace).List(api.ListEverything)
	return CreateEventList(FillEventsType(events.Items), dsQuery)
}

// Based on event Reason fills event Type in order to allow correct filtering by Type.
func FillEventsType(events []v1.Event) []v1.Event {
	for i := range events {
		// Fill in only events with empty type.
		if len(events[i].Type) == 0 {
			if isFailedReason(events[i].Reason, FailedReasonPartials...) {
				events[i].Type = v1.EventTypeWarning
			} else {
				events[i].Type = v1.EventTypeNormal
			}
		}
	}

	return events
}

// GetPodsEvents gets events targeting given list of pods.
func GetPodsEvents(client client.Interface, namespace string, pods []v1.Pod) ([]v1.Event, error) {
	nsQuery := common.NewSameNamespaceQuery(namespace)
	if namespace == v1.NamespaceAll {
		nsQuery = common.NewNamespaceQuery()
	}

	channels := &common.ResourceChannels{
		EventList: common.GetEventListChannel(client, nsQuery),
	}

	eventList := <-channels.EventList.List
	if err := <-channels.EventList.Error; err != nil {
		return nil, err
	}

	events := filterEventsByPodsUID(eventList.Items, pods)

	return events, nil
}

// GetPodEvents gets pods events associated to pod name and namespace
func GetPodEvents(client client.Interface, namespace, podName string) ([]v1.Event, error) {

	channels := &common.ResourceChannels{
		PodList:   common.GetPodListChannel(client, common.NewSameNamespaceQuery(namespace)),
		EventList: common.GetEventListChannel(client, common.NewSameNamespaceQuery(namespace)),
	}

	podList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	eventList := <-channels.EventList.List
	if err := <-channels.EventList.Error; err != nil {
		return nil, err
	}

	l := make([]v1.Pod, 0)
	for _, pi := range podList.Items {
		if pi.Name == podName {
			l = append(l, pi)
		}
	}

	events := filterEventsByPodsUID(eventList.Items, l)
	return FillEventsType(events), nil
}

// GetNodeEvents gets events associated to node with given name.
func GetNodeEvents(client client.Interface, dsQuery *dataselect.DataSelectQuery, nodeName string) (common.EventList, error) {
	eventList := common.EventList{
		Events: make([]common.Event, 0),
	}

	scheme := runtime.NewScheme()
	groupVersion := schema.GroupVersion{Group: "", Version: "v1"}
	scheme.AddKnownTypes(groupVersion, &v1.Node{})

	mc := client.CoreV1().Nodes()
	node, err := mc.Get(nodeName, metaV1.GetOptions{})
	if err != nil {
		return eventList, err
	}

	events, err := client.CoreV1().Events(v1.NamespaceAll).Search(scheme, node)
	if err != nil {
		return eventList, err
	}

	return CreateEventList(FillEventsType(events.Items), dsQuery)
}

// GetResourceEvents gets events associated to specified resource.
func GetResourceEvents(client client.Interface, dsQuery *dataselect.DataSelectQuery, namespace, name string) (
	*common.EventList, error) {
	resourceEvents, err := GetEvents(client, namespace, name)
	if err != nil {
		return nil, err
	}

	events, err := CreateEventList(resourceEvents, dsQuery)
	return &events, err
}

// CreateEventList converts array of api events to common EventList structure
func CreateEventList(events []v1.Event, dsQuery *dataselect.DataSelectQuery) (common.EventList, error) {
	eventList := &common.EventList{
		ListMeta: dataselect.NewListMeta(),
		Events:   make([]common.Event, 0),
	}
	err := dataselect.ToResourceList(
		eventList,
		events,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return *eventList, err
}
