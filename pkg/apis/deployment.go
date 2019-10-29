package apis

import (
	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ContainerUpdateInput struct {
	// required: true
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
}

type PodUpdateInput struct {
	K8sNamespaceResourceCreateInput

	InitContainers []ContainerUpdateInput `json:"initContainers,omitempty"`
	Containers     []ContainerUpdateInput `json:"containers,omitempty"`
	RestartPolicy  v1.RestartPolicy       `json:"restartPolicy,omitempty"`
	DNSPolicy      v1.DNSPolicy           `json:"dnsPolicy,omitempty"`
}

type DeploymentUpdateInput struct {
	Replicas *int32 `json:"replicas"`
	PodUpdateInput
}

type StatefulsetUpdateInput struct {
	Replicas *int32 `json:"replicas"`
	PodUpdateInput
}

type ContainerImage struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

// Deployment is a presentation layer view of kubernetes Deployment resource. This means
// it is Deployment plus additional augmented data we can get from other sources
// (like services that target the same pods)
type Deployment struct {
	ObjectMeta
	TypeMeta

	// Aggregate information about pods belonging to this deployment
	Pods PodInfo `json:"podsInfo"`

	Replicas *int32 `json:"replicas"`

	// Container images of the Deployment
	ContainerImages []ContainerImage `json:"containerImages"`

	// Init Container images of deployment
	InitContainerImages []ContainerImage `json:"initContainerImages"`

	Status   string            `json:"status"`
	Selector map[string]string `json:"selector"`
}

type StatusInfo struct {
	// Total number of desired replicas on the deployment
	Replicas int32 `json:"replicas"`

	// Number of non-terminated pods that have the desired template spec
	Updated int32 `json:"updated"`

	// Number of available pods (ready for at least minReadySeconds)
	// targeted by this deployment
	Available int32 `json:"available"`

	// Total number of unavailable pods targeted by this deployment.
	Unavailable int32 `json:"unavailable"`
}

// RollingUpdateStrategy is behavior of a rolling update. See RollingUpdateDeployment K8s object.
type RollingUpdateStrategy struct {
	MaxSurge       *intstr.IntOrString `json:"maxSurge"`
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable"`
}

// DeploymentDetail is a presentation layer view of Kubernetes Deployment resource.
type DeploymentDetail struct {
	Deployment
	// Detailed information about Pods belonging to this Deployment.
	PodList []Pod `json:"pods"`

	ServiceList []Service `json:"services"`

	// Status information on the deployment
	StatusInfo `json:"statusInfo"`

	// The deployment strategy to use to replace existing pods with new ones.
	// Valid options: Recreate, RollingUpdate
	Strategy apps.DeploymentStrategyType `json:"strategy"`

	// Min ready seconds
	MinReadySeconds int32 `json:"minReadySeconds"`

	// Rolling update strategy containing maxSurge and maxUnavailable
	RollingUpdateStrategy *RollingUpdateStrategy `json:"rollingUpdateStrategy,omitempty"`

	// RepliaSetList containing old replica sets from the deployment
	OldReplicaSetList []ReplicaSet `json:"oldReplicaSetList"`

	// New replica set used by this deployment
	NewReplicaSet ReplicaSet `json:"newReplicaSet"`

	// Optional field that specifies the number of old Replica Sets to retain to allow rollback.
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit"`

	// List of events related to this Deployment
	EventList []Event `json:"events"`

	// List of Horizontal Pod AutoScalers targeting this Deployment
	//HorizontalPodAutoscalerList hpa.HorizontalPodAutoscalerList `json:"horizontalPodAutoscalerList"`
}
