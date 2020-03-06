package statefulset

import (
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/event"
)

func getStatus(list []*apps.StatefulSet, pods []*v1.Pod, events []*v1.Event) common.ResourceStatus {
	info := common.ResourceStatus{}
	if list == nil {
		return info
	}

	for _, ss := range list {
		matchingPods := common.FilterPodsByControllerRef(ss, pods)
		podInfo := common.GetPodInfo(ss.Status.Replicas, ss.Spec.Replicas, matchingPods)
		warnings := event.GetPodsEventWarnings(events, matchingPods)

		if len(warnings) > 0 {
			info.Failed++
		} else if podInfo.Pending > 0 {
			info.Pending++
		} else {
			info.Running++
		}
	}

	return info
}
