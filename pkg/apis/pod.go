package apis

import (
	"k8s.io/api/core/v1"
)

// Pod is a presentation layer view of Pod resource. This means it is Pod plus additional augmented data
// we can get from other sources (like services that target it).
type Pod struct {
	ObjectMeta
	TypeMeta

	// More info on pod status
	PodStatus

	PodIP string `json:"podIP"`
	// Count of containers restarts
	RestartCount int32 `json:"restartCount"`

	// Pod warning events
	Warnings []Event `json:"warnings"`

	// Name of the Node this pod runs on
	NodeName string `json:"nodeName"`

	QOSClass       string      `json:"qosClass"`
	Containers     []Container `json:"containers"`
	InitContainers []Container `json:"initContainers"`
}

type PodStatus struct {
	Status          string              `json:"status"`
	PodPhase        v1.PodPhase         `json:"podPhase"`
	ContainerStates []v1.ContainerState `json:"containerStates"`
}

type PodDetail struct {
	Pod
	Conditions                []Condition             `json:"conditions"`
	Events                    []Event                 `json:"events"`
	PersistentvolumeclaimList []PersistentVolumeClaim `json:"persistentVolumeClaims"`
}

// Container represents a docker/rkt/etc. container that lives in a pod.
type Container struct {
	// Name of the container.
	Name string `json:"name"`

	// Image URI of the container.
	Image string `json:"image"`

	// List of environment variables.
	Env []EnvVar `json:"env"`

	// Commands of the container
	Commands []string `json:"commands"`

	// Command arguments
	Args []string `json:"args"`
}

// EnvVar represents an environment variable of a container.
type EnvVar struct {
	// Name of the variable.
	Name string `json:"name"`

	// Value of the variable. May be empty if value from is defined.
	Value string `json:"value"`

	// Defined for derived variables. If non-null, the value is get from the reference.
	// Note that this is an API struct. This is intentional, as EnvVarSources are plain struct
	// references.
	ValueFrom *v1.EnvVarSource `json:"valueFrom"`
}

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
	if info.Current == 0 {
		// delete
		return string(v1.PodPending)
	}
	if info.Failed > 0 {
		return string(v1.PodFailed)
	}
	if info.Pending > 0 {
		return string(v1.PodPending)
	}
	if info.Succeeded == *info.Desired {
		return string(v1.PodSucceeded)
	}
	if info.Running == *info.Desired {
		return string(v1.PodRunning)
	}
	return string(v1.PodUnknown)
}
