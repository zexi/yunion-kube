package apis

import (
	"k8s.io/api/core/v1"
)

// Namespace is a presentation layer view of Kubernetes namespaces. This means it is namespace plus
// additional augmented data we can get from other sources.
type Namespace struct {
	ObjectMeta
	TypeMeta

	Phase v1.NamespacePhase `json:"status"`
}

// NamespaceDetail is a presentation layer view of Kubernetes Namespace resource. This means it is Namespace plus
// additional augmented data we can get from other sources.
type NamespaceDetail struct {
	Namespace

	// Events is list of events associated to the namespace.
	EventList []Event `json:"events"`

	// ResourceQuotaList is list of resource quotas associated to the namespace
	ResourceQuotaList []ResourceQuotaDetail `json:"resourceQuotas"`

	// ResourceLimits is list of limit ranges associated to the namespace
	ResourceLimits []LimitRangeItem `json:"resourceLimits"`
}

type NamespaceCreateInput struct {
	K8sClusterResourceCreateInput
}
