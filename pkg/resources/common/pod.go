package common

import (
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "yunion.io/x/yunion-kube/pkg/api"
)

// FilterPodsByControllerResource returns a subset of pods controlled by given deployment.
func FilterDeploymentPodsByOwnerReference(deployment *apps.Deployment, allRS []*apps.ReplicaSet,
	allPods []*v1.Pod) []*v1.Pod {
	var matchingPods []*v1.Pod
	for _, rs := range allRS {
		if metav1.IsControlledBy(rs, deployment) {
			matchingPods = append(matchingPods, FilterPodsByControllerRef(rs, allPods)...)
		}
	}

	return matchingPods
}

// FilterPodsByControllerRef returns a subset of pods controlled by given controller resource, excluding deployments.
func FilterPodsByControllerRef(owner metav1.Object, allPods []*v1.Pod) []*v1.Pod {
	var matchingPods []*v1.Pod
	for _, pod := range allPods {
		if metav1.IsControlledBy(pod, owner) {
			matchingPods = append(matchingPods, pod)
		}
	}
	return matchingPods
}

func FilterPodsForJob(job *batch.Job, pods []*v1.Pod) []*v1.Pod {
	result := make([]*v1.Pod, 0)
	for _, pod := range pods {
		if pod.Namespace == job.Namespace && pod.Labels["controller-uid"] ==
			job.Spec.Selector.MatchLabels["controller-uid"] {
			result = append(result, pod)
		}
	}

	return result
}

// GetContainerImages returns container image strings from the given pod spec.
func GetContainerImages(podTemplate *v1.PodSpec) []api.ContainerImage {
	containerImages := []api.ContainerImage{}
	for _, container := range podTemplate.Containers {
		containerImages = append(containerImages, api.ContainerImage{
			Name:  container.Name,
			Image: container.Image,
		})
	}
	return containerImages
}

// GetInitContainerImages returns init container image strings from the given pod spec.
func GetInitContainerImages(podTemplate *v1.PodSpec) []api.ContainerImage {
	initContainerImages := []api.ContainerImage{}
	for _, initContainer := range podTemplate.InitContainers {
		initContainerImages = append(initContainerImages, api.ContainerImage{
			Name:  initContainer.Name,
			Image: initContainer.Image})
	}
	return initContainerImages
}

// GetContainerNames returns the container image name without the version number from the given pod spec.
func GetContainerNames(podTemplate *v1.PodSpec) []string {
	var containerNames []string
	for _, container := range podTemplate.Containers {
		containerNames = append(containerNames, container.Name)
	}
	return containerNames
}

// GetInitContainerNames returns the init container image name without the version number from the given pod spec.
func GetInitContainerNames(podTemplate *v1.PodSpec) []string {
	var initContainerNames []string
	for _, initContainer := range podTemplate.InitContainers {
		initContainerNames = append(initContainerNames, initContainer.Name)
	}
	return initContainerNames
}

// EqualIgnoreHash returns true if two given podTemplateSpec are equal, ignoring the diff in value of Labels[pod-template-hash]
// We ignore pod-template-hash because the hash result would be different upon podTemplateSpec API changes
// (e.g. the addition of a new field will cause the hash code to change)
// Note that we assume input podTemplateSpecs contain non-empty labels
func EqualIgnoreHash(template1, template2 v1.PodTemplateSpec) bool {
	// First, compare template.Labels (ignoring hash)
	labels1, labels2 := template1.Labels, template2.Labels
	if len(labels1) > len(labels2) {
		labels1, labels2 = labels2, labels1
	}
	// We make sure len(labels2) >= len(labels1)
	for k, v := range labels2 {
		if labels1[k] != v && k != apps.DefaultDeploymentUniqueLabelKey {
			return false
		}
	}
	// Then, compare the templates without comparing their labels
	template1.Labels, template2.Labels = nil, nil
	return equality.Semantic.DeepEqual(template1, template2)
}
