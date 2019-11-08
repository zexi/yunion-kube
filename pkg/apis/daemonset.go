package apis

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DaemonSet plus zero or more Kubernetes services that target the Daemon Set.
type DaemonSet struct {
	ObjectMeta
	TypeMeta
	// Aggregate information about pods belonging to this deployment
	Pods   PodInfo `json:"podsInfo"`
	Status string  `json:"status"`

	ContainerImages     []ContainerImage `json:"containerImages"`
	InitContainerImages []ContainerImage `json:"initContainerImages"`
	Selector *v1.LabelSelector `json:"labelSelector"`
}
