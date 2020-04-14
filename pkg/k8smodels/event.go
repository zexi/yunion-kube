package k8smodels

import (
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
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

// FailedReasonPartials  is an array of partial strings to correctly filter warning events.
// Have to be lower case for correct case insensitive comparison.
// Based on k8s official events reason file:
// https://github.com/kubernetes/kubernetes/blob/886e04f1fffbb04faf8a9f9ee141143b2684ae68/pkg/kubelet/events/event.go
// Partial strings that are not in event.go file are added in order to support
// older versions of k8s which contained additional event reason messages.
var FailedReasonPartials = []string{"failed", "err", "exceeded", "invalid", "unhealthy",
	"mismatch", "insufficient", "conflict", "outof", "nil", "backoff"}

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
		KindName:     apis.KindNameEvent,
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
			if m.isFailedReason(e.Reason, FailedReasonPartials...) {
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
	return m.GetAPIEvents(obj.GetCluster(), res)
}

func (m SEventManager) GetNamespaceEvents(cluster model.ICluster, ns string) ([]*apis.Event, error) {
	events, err := m.GetRawEvents(cluster, ns)
	if err != nil {
		return nil, err
	}
	events = m.fillEventsType(events)
	return m.GetAPIEvents(cluster, events)
}

func (m SEventManager) GetRawEventsByUID(cluster model.ICluster, uId types.UID) ([]*v1.Event, error) {
	events, err := m.GetAllRawEvents(cluster)
	if err != nil {
		return nil, err
	}
	return m.FilterEventsByUID(events, uId), nil
}

func (m SEventManager) GetRawEventsByKindName(cluster model.ICluster, kind, namespace, name string) ([]*v1.Event, error) {
	events, err := m.GetAllRawEvents(cluster)
	if err != nil {
		return nil, err
	}
	return m.FilterEventsByKindName(events, kind, namespace, name), nil
}

func (m SEventManager) ListItemFilter(ctx *model.RequestContext, q model.IQuery, query *apis.EventListInput) (model.IQuery, error) {
	q, err := m.SK8SNamespaceResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SNamespaceBase)
	if err != nil {
		return q, err
	}
	if query.ListInputOwner.ShouldDo() {
		q.AddFilter(m.ListOwnerFilter(query.ListInputOwner, query.Namespace))
	}
	return q, nil
}

func (m SEventManager) ListOwnerFilter(input apis.ListInputOwner, namespace string) model.QueryFilter {
	return func(obj model.IK8SModel) (bool, error) {
		event := obj.(*SEvent).GetRawEvent()
		man := model.GetK8SModelManagerByKind(input.OwnerKind)
		if man == nil {
			return false, httperrors.NewNotFoundError("Not found owner_kind %s", input.OwnerKind)
		}
		ownerModel, err := model.NewK8SModelObjectByName(man, obj.GetCluster(), namespace, input.OwnerName)
		if err != nil {
			return false, err
		}
		return model.IsEventOwner(ownerModel, event)
	}
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
	return m.GetAPIEvents(cluster, es)
}

func (m SEventManager) GetEventsByUID(cluster model.ICluster, uId types.UID) ([]*apis.Event, error) {
	res, err := m.GetRawEventsByUID(cluster, uId)
	if err != nil {
		return nil, err
	}
	return m.GetAPIEvents(cluster, res)
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

func (m SEventManager) FilterEventsByKindName(events []*v1.Event, kind, namespace, name string) []*v1.Event {
	result := make([]*v1.Event, 0)
	for _, e := range events {
		iobj := e.InvolvedObject
		if iobj.Kind == kind && iobj.Namespace == namespace && iobj.Name == name {
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

func (m *SEventManager) GetAPIEvents(cluster model.ICluster, events []*v1.Event) ([]*apis.Event, error) {
	ret := make([]*apis.Event, 0)
	if err := ConvertRawToAPIObjects(m, cluster, events, &ret); err != nil {
		return nil, err
	}
	s := &eventLastTimestampSorter{
		events: ret,
	}
	sort.Sort(s)
	return s.events, nil
}

type eventLastTimestampSorter struct {
	events []*apis.Event
}

func (s *eventLastTimestampSorter) Less(i, j int) bool {
	e1 := s.events[i]
	e2 := s.events[j]
	return e1.LastSeen.Before(&e2.LastSeen)
}

func (s *eventLastTimestampSorter) Len() int {
	return len(s.events)
}

func (s *eventLastTimestampSorter) Swap(i, j int) {
	s.events[i], s.events[j] = s.events[j], s.events[i]
}

func (obj SEvent) GetRawEvent() *v1.Event {
	return obj.GetK8SObject().(*v1.Event)
}

func (obj SEvent) GetAPIObject() (*apis.Event, error) {
	e := obj.GetRawEvent()
	return &apis.Event{
		ObjectMeta:          obj.GetObjectMeta(),
		TypeMeta:            obj.GetTypeMeta(),
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
	}, nil
}

func (obj SEvent) GetAPIDetailObject() (*apis.Event, error) {
	return obj.GetAPIObject()
}
