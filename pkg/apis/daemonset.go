package apis

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	DaemonSetStatusObservedWaiting = "ObservedWaiting"
	DaemonSetStatusPodReadyWaiting = "PodReadyWaiting"
	DaemonSetStatusUpdateWaiting   = "UpdateWaiting"
)

// DaemonSet plus zero or more Kubernetes services that target the Daemon Set.
type DaemonSet struct {
	ObjectMeta
	TypeMeta

	// Aggregate information about pods belonging to this deployment
	PodInfo PodInfo `json:"podsInfo"`

	ContainerImages     []ContainerImage  `json:"containerImages"`
	InitContainerImages []ContainerImage  `json:"initContainerImages"`
	Selector            *v1.LabelSelector `json:"labelSelector"`

	DaemonSetStatus
}

type DaemonSetStatus struct {
	Status string `json:"status"`
}

type DaemonSetDetails struct {
	DaemonSet

	Events []*Event `json:"events"`
}
