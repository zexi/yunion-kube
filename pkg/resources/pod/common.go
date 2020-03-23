package pod

import (
	"k8s.io/api/core/v1"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

func ToPod(
	pod *v1.Pod,
	warnings []api.Event,
	cfgs []*v1.ConfigMap,
	secrets []*v1.Secret,
	cluster api.ICluster,
) api.Pod {
	podDetail := api.Pod{
		ObjectMeta:     api.NewObjectMeta(pod.ObjectMeta, cluster),
		TypeMeta:       api.NewTypeMeta(pod.TypeMeta),
		Warnings:       warnings,
		PodStatus:      getPodStatus(pod),
		RestartCount:   getRestartCount(pod),
		PodIP:          pod.Status.PodIP,
		QOSClass:       string(pod.Status.QOSClass),
		Containers:     extractContainerInfo(pod.Spec.Containers, pod, cfgs, secrets),
		InitContainers: extractContainerInfo(pod.Spec.InitContainers, pod, cfgs, secrets),
	}
	return podDetail
}

func GetRestartCount(pod *v1.Pod) int32 {
	return getRestartCount(pod)
}

func getRestartCount(pod *v1.Pod) int32 {
	var restartCount int32 = 0
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restartCount += containerStatus.RestartCount
	}
	return restartCount
}

func getPodStatus(pod *v1.Pod) api.PodStatus {
	var states []v1.ContainerState
	for _, containerStatus := range pod.Status.ContainerStatuses {
		states = append(states, containerStatus.State)
	}
	return api.PodStatus{
		PodStatusV2:     *getters.GetPodStatus(pod),
		PodPhase:        pod.Status.Phase,
		ContainerStates: states,
	}
}

func getPodStatusPhase(pod *v1.Pod) v1.PodPhase {
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
	return v1.PodPending
}

func getPodConditions(pod v1.Pod) []api.Condition {
	var conditions []api.Condition
	for _, condition := range pod.Status.Conditions {
		conditions = append(conditions, api.Condition{
			Type:               string(condition.Type),
			Status:             condition.Status,
			LastProbeTime:      condition.LastProbeTime,
			LastTransitionTime: condition.LastTransitionTime,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	return conditions
}
