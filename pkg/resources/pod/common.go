package pod

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func ToPod(pod v1.Pod) Pod {
	podDetail := Pod{
		ObjectMeta:   api.NewObjectMeta(pod.ObjectMeta),
		TypeMeta:     api.NewTypeMeta(api.ResourceKindPod),
		PodStatus:    getPodStatus(pod),
		RestartCount: getRestartCount(pod),
		NodeName:     pod.Spec.NodeName,
		PodIP:        pod.Status.PodIP,
	}
	return podDetail
}

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
