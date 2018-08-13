package pod

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

func getRestartCount(pod v1.Pod) int32 {
	var restartCount int32 = 0
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restartCount += containerStatus.RestartCount
	}
	return restartCount
}

func getPodStatus(pod v1.Pod) PodStatus {
	var states []v1.ContainerState
	for _, containerStatus := range pod.Status.ContainerStatuses {
		states = append(states, containerStatus.State)
	}
	return PodStatus{
		Status:          string(getPodStatusPhase(pod)),
		PodPhase:        pod.Status.Phase,
		ContainerStates: states,
	}
}

func getPodStatusPhase(pod v1.Pod) v1.PodPhase {
	// for terminated pods that filed
	if pod.Status.Phase == v1.PodFailed {
		return v1.PodFailed
	}

	// for successfully terminated pods
	if pod.Status.Phase == v1.PodSucceeded {
		return v1.PodSucceeded
	}

	ready := false
	initialized := false
	for _, c := range pod.Status.Conditions {
		if c.Type == v1.PodReady {
			ready = c.Status == v1.ConditionTrue
		}
		if c.Type == v1.PodInitialized {
			initialized = c.Status == v1.ConditionTrue
		}
	}

	if initialized && ready && pod.Status.Phase == v1.PodRunning {
		return v1.PodRunning
	}

	// Unknown?
	//return v1.PodPending
	return pod.Status.Phase
}

type PodCell v1.Pod

func (p PodCell) GetProperty(name dataselect.PropertyName) dataselect.ComparableValue {
	switch name {
	case dataselect.NameProperty:
		return dataselect.StdComparableString(p.ObjectMeta.Name)
	case dataselect.CreationTimestampProperty:
		return dataselect.StdComparableTime(p.ObjectMeta.CreationTimestamp.Time)
	case dataselect.NamespaceProperty:
		return dataselect.StdComparableString(p.ObjectMeta.Namespace)
	case dataselect.StatusProperty:
		return dataselect.StdComparableString(p.Status.Phase)
	default:
		return nil
	}
}

func toCells(std []v1.Pod) []dataselect.DataCell {
	cells := make([]dataselect.DataCell, len(std))
	for i := range std {
		cells[i] = PodCell(std[i])
	}
	return cells
}

func fromCells(cells []dataselect.DataCell) []v1.Pod {
	std := make([]v1.Pod, len(cells))
	for i := range std {
		std[i] = v1.Pod(cells[i].(PodCell))
	}
	return std
}
