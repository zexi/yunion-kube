package event

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
	"yunion.io/x/yunion-kube/pkg/client"
)

// GetEvents gets events associated to resource with given name.
func GetEvents(indexer *client.CacheFactory, namespace, resourceName string) ([]*v1.Event, error) {
	channels := &common.ResourceChannels{
		EventList: common.GetEventListChannelWithOptions(
			indexer,
			common.NewSameNamespaceQuery(namespace),
			labels.Everything()),
	}

	eventList := <-channels.EventList.List
	if err := <-channels.EventList.Error; err != nil {
		return nil, err
	}
	filterEventList := make([]*v1.Event, 0)
	for _, e := range eventList {
		if e.InvolvedObject.Name == resourceName {
			filterEventList = append(filterEventList, e)
		}
	}

	return FillEventsType(filterEventList), nil
}

// GetNamespaceEvents gets events associated to a namespace with given name.
func GetNamespaceEvents(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace string) (*common.EventList, error) {
	events, _ := indexer.EventLister().Events(namespace).List(labels.Everything())
	return CreateEventList(FillEventsType(events), dsQuery, cluster)
}

// Based on event Reason fills event Type in order to allow correct filtering by Type.
func FillEventsType(events []*v1.Event) []*v1.Event {
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
func GetPodsEvents(indexer *client.CacheFactory, namespace string, pods []*v1.Pod) ([]*v1.Event, error) {
	nsQuery := common.NewSameNamespaceQuery(namespace)
	if namespace == v1.NamespaceAll {
		nsQuery = common.NewNamespaceQuery()
	}

	channels := &common.ResourceChannels{
		EventList: common.GetEventListChannel(indexer, nsQuery),
	}

	eventList := <-channels.EventList.List
	if err := <-channels.EventList.Error; err != nil {
		return nil, err
	}

	events := filterEventsByPodsUID(eventList, pods)

	return events, nil
}

// GetPodEvents gets pods events associated to pod name and namespace
func GetPodEvents(indexer *client.CacheFactory, namespace, podName string) ([]*v1.Event, error) {
	channels := &common.ResourceChannels{
		PodList:   common.GetPodListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
		EventList: common.GetEventListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
	}

	podList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	eventList := <-channels.EventList.List
	if err := <-channels.EventList.Error; err != nil {
		return nil, err
	}

	l := make([]*v1.Pod, 0)
	for _, pi := range podList {
		if pi.Name == podName {
			l = append(l, pi)
		}
	}

	events := filterEventsByPodsUID(eventList, l)
	return FillEventsType(events), nil
}

// GetNodeEvents gets events associated to node with given name.
func GetNodeEvents(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, nodeID types.UID) (*common.EventList, error) {
	events, err := indexer.EventLister().Events("").List(labels.Everything())
	if err != nil {
		return nil, err
	}

	events = FilterEventsByUID(events, nodeID)

	return CreateEventList(FillEventsType(events), dsQuery, cluster)
}

// GetResourceEvents gets events associated to specified resource.
func GetResourceEvents(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace, name string) (
	*common.EventList, error) {
	resourceEvents, err := GetEvents(indexer, namespace, name)
	if err != nil {
		return nil, err
	}

	events, err := CreateEventList(resourceEvents, dsQuery, cluster)
	return events, err
}

// CreateEventList converts array of api events to common EventList structure
func CreateEventList(events []*v1.Event, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*common.EventList, error) {
	eventList := &common.EventList{
		BaseList: common.NewBaseList(cluster),
		Events:   make([]common.Event, 0),
	}
	err := dataselect.ToResourceList(
		eventList,
		events,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return eventList, err
}
