package api

import (
	"k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

// RbacRole provides the simplified, combined presentation layer view of Kubernetes' RBAC Roles and ClusterRoles.
// ClusterRoles will be referred to as Roles for the namespace "all namespaces".
type RbacRole struct {
	ObjectMeta
	TypeMeta

	Type string `json:"type"`

	Rules []rbac.PolicyRule `json:"rules"`

	AggregationRule *rbac.AggregationRule `json:"aggregationRule,omitempty"`
}

// RbacRoleBinding contains ClusterRoleBinding and RoleBinding
type RbacRoleBinding struct {
	ObjectMeta

	TypeMeta

	Type string `json:"type"`

	Subjects []rbac.Subject `json:"subjects,omitempty"`

	RoleRef rbac.RoleRef `json:"roleRef"`
}

// ClusterRole is a cluster level, logical grouping of PolicyRules that can be referenced as a unit by a RoleBinding or ClusterRoleBinding.
type ClusterRole struct {
	ObjectMeta
	TypeMeta

	Rules []rbac.PolicyRule `json:"rules"`

	AggregationRule *rbac.AggregationRule `json:"aggregationRule,omitempty"`
}

// ClusterRoleBinding references a ClusterRole, but not contain it.  It can reference a ClusterRole in the global namespace,
// and adds who information via Subject.
type ClusterRoleBinding struct {
	ObjectMeta

	TypeMeta

	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects []rbac.Subject `json:"subjects,omitempty"`

	// RoleRef can only reference a ClusterRole in the global namespace.
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	RoleRef rbac.RoleRef `json:"roleRef"`
}

// Role is a namespaced, logical grouping of PolicyRules that can be referenced as a unit by a RoleBinding.
type Role struct {
	ObjectMeta
	TypeMeta

	Rules []rbac.PolicyRule `json:"rules"`
}

// RoleBinding references a role, but does not contain it.  It can reference a Role in the same namespace or a ClusterRole in the global namespace.
// It adds who information via Subjects and namespace information by which namespace it exists in.  RoleBindings in a given
// namespace only have effect in that namespace.
type RoleBinding struct {
	ObjectMeta

	TypeMeta

	// Subjects holds references to the objects the role applies to.
	Subjects []rbac.Subject `json:"subjects,omitempty"`

	// RoleRef can reference a Role in the current namespace or a ClusterRole in the global namespace.
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	RoleRef rbac.RoleRef `json:"roleRef"`
}

type ServiceAccount struct {
	NamespaceResourceDetail
	// Secrets is the list of secrets allowed to be used by pods running using this ServiceAccount.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Secrets []v1.ObjectReference `json:"secrets,omitempty"`

	// ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images
	// in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets
	// can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet.
	// More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// AutomountServiceAccountToken indicates whether pods running as this service account should have an API token automatically mounted.
	// Can be overridden at the pod level.
	// +optional
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty"`
}
