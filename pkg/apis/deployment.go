package apis

import (
	"k8s.io/api/core/v1"
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
