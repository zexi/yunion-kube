package apis

import (
	"k8s.io/api/core/v1"
)

// ResourceQuotaDetail provides the presentation layer view of Kubernetes Resource Quotas resource.
type ResourceQuotaDetail struct {
	ObjectMeta
	TypeMeta

	// Scopes defines quota scopes
	Scopes []v1.ResourceQuotaScope `json:"scopes,omitempty"`

	// StatusList is a set of (resource name, Used, Hard) tuple.
	StatusList map[v1.ResourceName]ResourceStatus `json:"statuses,omitempty"`
}

// ResourceStatus provides the status of the resource defined by a resource quota.
type ResourceStatus struct {
	Used string `json:"used,omitempty"`
	Hard string `json:"hard,omitempty"`
}
