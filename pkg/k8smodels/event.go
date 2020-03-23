package k8smodels

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	"yunion.io/x/yunion-kube/pkg/resources/event"
)

var (
	EventManager *SEventManager
)

func init() {
	EventManager = &SEventManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SEvent{},
			"k8s_event",
			"k8s_events"),
	}
	EventManager.SetVirtualObject(EventManager)
}

type SEventManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SEvent struct {
	model.SK8SNamespaceResourceBase
}

func (m SEventManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameEvent,
		Object:       &v1.Event{},
	}
}

func (m SEventManager) GetRawEvents(cluster model.ICluster, ns string) ([]*v1.Event, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.EventLister().Events(ns).List(labels.Everything())
}

func (m SEventManager) GetAllRawEvents(cluster model.ICluster) ([]*v1.Event, error) {
	return m.GetRawEvents(cluster, v1.NamespaceAll)
}

// Returns true if reason string contains any partial string indicating that this may be a
// warning, false otherwise
func (m SEventManager) isFailedReason(reason string, partials ...string) bool {
	for _, partial := range partials {
		if strings.Contains(strings.ToLower(reason), partial) {
			return true
		}
	}

	return false
}

func (m SEventManager) fillEventsType(events []*v1.Event) []*v1.Event {
	for _, e := range events {
		// Fill in only events with empty type
		if len(e.Type) == 0 {
			if m.isFailedReason(e.Reason, event.FailedReasonPartials...) {
				e.Type = v1.EventTypeWarning
			} else {
				e.Type = v1.EventTypeNormal
			}
		}
	}
	return events
}

func (m SEventManager) GetRawEventsByResource(cluster model.ICluster, namespace string, resName string) ([]*v1.Event, error) {
	events, err := m.GetRawEvents(cluster, namespace)
	if err != nil {
		return nil, err
	}
	filtered := make([]*v1.Event, 0)
	for _, e := range events {
		if e.InvolvedObject.Name == resName {
			filtered = append(filtered, e)
		}
	}
	return m.fillEventsType(filtered), nil
}

func (m SEventManager) GetRawEventsByObject(cluster model.ICluster, obj runtime.Object) ([]*v1.Event, error) {
	return m.GetRawEventsByUID(cluster, obj.(metav1.Object).GetUID())
}

func (m SEventManager) GetEventsByObject(obj model.IK8SModel) ([]*apis.Event, error) {
	res, err := m.GetRawEventsByObject(obj.GetCluster(), obj.GetK8SObject())
	if err != nil {
		return nil, err
	}
	return m.GetAPIEvents(obj.GetCluster(), res), nil
}

func (m SEventManager) GetRawEventsByUID(cluster model.ICluster, uId types.UID) ([]*v1.Event, error) {
	events, err := m.GetAllRawEvents(cluster)
	if err != nil {
		return nil, err
	}
	return m.FilterEventsByUID(events, uId), nil
}

func (m SEventManager) GetRawEventsByPods(cluster model.ICluster, pods []*v1.Pod) ([]*v1.Event, error) {
	result := make([]*v1.Event, 0)
	podEventMap := make(map[types.UID]bool, 0)

	if len(pods) == 0 {
		return result, nil
	}

	for _, pod := range pods {
		podEventMap[pod.UID] = true
	}

	events, err := m.GetAllRawEvents(cluster)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if _, exists := podEventMap[event.InvolvedObject.UID]; exists {
			result = append(result, event)
		}
	}

	return result, nil
}

// Returns true if given pod is in state ready or succeeded, false otherwise
func isReadyOrSucceededPod(pod *v1.Pod) bool {
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

// Returns filtered list of event objects. Events list is filtered to get only events targeting
// pods on the list.
func (m SEventManager) filterEventsByPods(events []*v1.Event, pods []*v1.Pod) []*v1.Event {
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

// Removes duplicate strings from the slice
func (m SEventManager) removeDuplicates(slice []*v1.Event) []*v1.Event {
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

func (m SEventManager) GetRawWarningEventsByPods(cluster model.ICluster, pods []*v1.Pod) ([]*v1.Event, error) {
	podEvents, err := m.GetRawEventsByPods(cluster, pods)
	if err != nil {
		return nil, err
	}

	// Filter out only warning events
	events := m.FilterEventsByType(podEvents, v1.EventTypeWarning)
	failedPods := make([]*v1.Pod, 0)

	// Filter out ready and successful pods
	for _, pod := range pods {
		if !isReadyOrSucceededPod(pod) {
			failedPods = append(failedPods, pod)
		}
	}

	events = m.filterEventsByPods(events, failedPods)
	events = m.removeDuplicates(events)
	return events, nil
}

func (m SEventManager) GetWarningEventsByPods(cluster model.ICluster, pods []*v1.Pod) ([]*apis.Event, error) {
	es, err := m.GetRawWarningEventsByPods(cluster, pods)
	if err != nil {
		return nil, err
	}
	return m.GetAPIEvents(cluster, es), nil
}

func (m SEventManager) GetEventsByUID(cluster model.ICluster, uId types.UID) ([]*apis.Event, error) {
	res, err := m.GetRawEventsByUID(cluster, uId)
	if err != nil {
		return nil, err
	}
	return m.GetAPIEvents(cluster, res), nil
}

func (m SEventManager) FilterEventsByUID(events []*v1.Event, uid types.UID) []*v1.Event {
	result := make([]*v1.Event, 0)
	for _, e := range events {
		if e.InvolvedObject.UID == uid {
			result = append(result, e)
		}
	}
	return m.fillEventsType(result)
}

func (m SEventManager) FilterEventsByType(events []*v1.Event, eventType string) []*v1.Event {
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

func (m SEventManager) GetAPIEvents(cluster model.ICluster, events []*v1.Event) []*apis.Event {
	ret := make([]*apis.Event, len(events))
	for i := range events {
		ret[i] = m.GetAPIEvent(cluster, events[i])
	}
	return ret
}

func (m SEventManager) GetAPIEvent(cluster model.ICluster, e *v1.Event) *apis.Event {
	return &apis.Event{
		ObjectMeta:          apis.NewObjectMeta(e.ObjectMeta, cluster),
		TypeMeta:            apis.NewTypeMeta(e.TypeMeta),
		Message:             e.Message,
		SourceComponent:     e.Source.Component,
		SourceHost:          e.Source.Host,
		SubObject:           e.InvolvedObject.FieldPath,
		Count:               e.Count,
		FirstSeen:           e.FirstTimestamp,
		LastSeen:            e.LastTimestamp,
		Reason:              e.Reason,
		Type:                e.Type,
		InvolvedObject:      e.InvolvedObject,
		Source:              e.Source,
		Series:              e.Series,
		Action:              e.Action,
		Related:             e.Related,
		ReportingController: e.ReportingController,
		ReportingInstance:   e.ReportingInstance,
	}
}
