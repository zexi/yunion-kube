package common

import (
	api "k8s.io/api/core/v1"
)

// PodInfo represents aggregate information about controller's pods.
type PodInfo struct {
	// Number of pods that are created.
	Current int32 `json:"current"`

	// Number of pods that are desired.
	Desired *int32 `json:"desired,omitempty"`

	// Number of pods that are currently running.
	Running int32 `json:"running"`

	// Number of pods that are currently waiting.
	Pending int32 `json:"pending"`

	// Number of pods that are failed.
	Failed int32 `json:"failed"`

	// Number of pods that are succeeded.
	Succeeded int32 `json:"succeeded"`

	// Unique warning messages related to pods in this resource.
	Warnings []Event `json:"warnings"`
}

func (info PodInfo) GetStatus() string {
	if info.Failed > 0 {
		return string(api.PodFailed)
	}
	if info.Pending > 0 {
		return string(api.PodPending)
	}
	if info.Succeeded == *info.Desired {
		return string(api.PodSucceeded)
	}
	if info.Running == *info.Desired {
		return string(api.PodRunning)
	}
	return string(api.PodPending)
}

// GetPodInfo returns aggregate information about a group of pods.
func GetPodInfo(current int32, desired *int32, pods []api.Pod) PodInfo {
	result := PodInfo{
		Current:  current,
		Desired:  desired,
		Warnings: make([]Event, 0),
	}

	for _, pod := range pods {
		switch pod.Status.Phase {
		case api.PodRunning:
			result.Running++
		case api.PodPending:
			result.Pending++
		case api.PodFailed:
			result.Failed++
		case api.PodSucceeded:
			result.Succeeded++
		}
	}

	return result
}
