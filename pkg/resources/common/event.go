package common

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/yunion-kube/pkg/apis"
)

// EventList is an events response structure.
type EventList struct {
	*BaseList
	// List of events from given namespace.
	Events []apis.Event `json:"events"`
}

func (l *EventList) Append(obj interface{}) {
	event := obj.(*v1.Event)
	l.Events = append(l.Events, apis.Event{
		ObjectMeta:      apis.NewObjectMeta(event.ObjectMeta, l.GetCluster()),
		TypeMeta:        apis.NewTypeMeta(event.TypeMeta),
		Message:         event.Message,
		SourceComponent: event.Source.Component,
		SourceHost:      event.Source.Host,
		SubObject:       event.InvolvedObject.FieldPath,
		Count:           event.Count,
		FirstSeen:       event.FirstTimestamp,
		LastSeen:        event.LastTimestamp,
		Reason:          event.Reason,
		Type:            event.Type,
		Source:          event.Source,
		InvolvedObject:  event.InvolvedObject,
	})
}
