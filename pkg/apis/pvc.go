package apis

import (
	"k8s.io/api/core/v1"
)

// PersistentVolumeClaim provides the simplified presentation layer view of Kubernetes Persistent Volume Claim resource.
type PersistentVolumeClaim struct {
	ObjectMeta
	TypeMeta
	Status       string                          `json:"status"`
	Volume       string                          `json:"volume"`
	Capacity     v1.ResourceList                 `json:"capacity"`
	AccessModes  []v1.PersistentVolumeAccessMode `json:"accessModes"`
	StorageClass *string                         `json:"storageClass"`
	// Deprecated
	MountedBy []string `json:"mountedBy"`
}

// PersistentVolumeClaimDetail provides the presentation layer view of Kubernetes Persistent Volume Claim resource.
type PersistentVolumeClaimDetail struct {
	PersistentVolumeClaim
	Pods []*Pod `json:"pods"`
}

type PersistentVolumeClaimCreateInput struct {
	K8sNamespaceResourceCreateInput
	Size         string `json:"size"`
	StorageClass string `json:"storageClass"`
}

type PersistentVolumeClaimListInput struct {
	ListInputK8SNamespaceBase
	Unused *bool `json:"unused"`
}
