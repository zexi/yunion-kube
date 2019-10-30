package event

import (
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

// FailedReasonPartials  is an array of partial strings to correctly filter warning events.
// Have to be lower case for correct case insensitive comparison.
// Based on k8s official events reason file:
// https://github.com/kubernetes/kubernetes/blob/886e04f1fffbb04faf8a9f9ee141143b2684ae68/pkg/kubelet/events/event.go
// Partial strings that are not in event.go file are added in order to support
// older versions of k8s which contained additional event reason messages.
var FailedReasonPartials = []string{"failed", "err", "exceeded", "invalid", "unhealthy",
	"mismatch", "insufficient", "conflict", "outof", "nil", "backoff"}

// GetPodsEventWarnings returns warning pod events by filtering out events targeting only given pods
func GetPodsEventWarnings(events []*v1.Event, pods []*v1.Pod) []api.Event {
	result := make([]api.Event, 0)

	// Filter out only warning events
	events = getWarningEvents(events)
	failedPods := make([]*v1.Pod, 0)

	// Filter out ready and successful pods
	for _, pod := range pods {
		if !isReadyOrSucceeded(pod) {
			failedPods = append(failedPods, pod)
		}
	}

	// Filter events by failed pods UID
	events = filterEventsByPodsUID(events, failedPods)
	events = removeDuplicates(events)

	for _, event := range events {
		result = append(result, api.Event{
			Message: event.Message,
			Reason:  event.Reason,
			Type:    event.Type,
		})
	}

	return result
}

// Returns filtered list of event objects. Events list is filtered to get only events targeting
// pods on the list.
func filterEventsByPodsUID(events []*v1.Event, pods []*v1.Pod) []*v1.Event {
	result := make([]*v1.Event, 0)
	podEventMap := make(map[types.UID]bool, 0)

	if len(pods) == 0 || len(events) == 0 {
		return result
	}

	for _, pod := range pods {
		podEventMap[pod.UID] = true
	}

	for _, event := range events {
		if _, exists := podEventMap[event.InvolvedObject.UID]; exists {
			result = append(result, event)
		}
	}

	return result
}

func FilterEventsByUID(events []*v1.Event, uid types.UID) []*v1.Event {
	result := make([]*v1.Event, 0)
	for _, e := range events {
		if e.InvolvedObject.UID == uid {
			result = append(result, e)
		}
	}
	return result
}

// Returns filtered list of event objects.
// Event list object is filtered to get only warning events.
func getWarningEvents(events []*v1.Event) []*v1.Event {
	return filterEventsByType(FillEventsType(events), v1.EventTypeWarning)
}

// Filters kubernetes API event objects based on event type.
// Empty string will return all events.
func filterEventsByType(events []*v1.Event, eventType string) []*v1.Event {
	if len(eventType) == 0 || len(events) == 0 {
		return events
	}

	result := make([]*v1.Event, 0)
	for _, event := range events {
		if event.Type == eventType {
			result = append(result, event)
		}
	}

	return result
}

// Returns true if reason string contains any partial string indicating that this may be a
// warning, false otherwise
func isFailedReason(reason string, partials ...string) bool {
	for _, partial := range partials {
		if strings.Contains(strings.ToLower(reason), partial) {
			return true
		}
	}

	return false
}

// Removes duplicate strings from the slice
func removeDuplicates(slice []*v1.Event) []*v1.Event {
	visited := make(map[string]bool, 0)
	result := make([]*v1.Event, 0)

	for _, elem := range slice {
		if !visited[elem.Reason] {
			visited[elem.Reason] = true
			result = append(result, elem)
		}
	}

	return result
}

// Returns true if given pod is in state ready or succeeded, false otherwise
func isReadyOrSucceeded(pod *v1.Pod) bool {
	if pod.Status.Phase == v1.PodSucceeded {
		return true
	}
	if pod.Status.Phase == v1.PodRunning {
		for _, c := range pod.Status.Conditions {
			if c.Type == v1.PodReady {
				if c.Status == v1.ConditionFalse {
					return false
				}
			}
		}

		return true
	}

	return false
}
