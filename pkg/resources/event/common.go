package event

import (
	"k8s.io/api/core/v1"
)

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
