package api

import (
	rbac "k8s.io/api/rbac/v1"
)

type RoleRef struct {
	Kind string `json:"kind"`
	Id   string `json:"id"`
	// Name should set by backend
	Name string `json:"-"`
}

func (ref RoleRef) ToRoleRef() rbac.RoleRef {
	apiGroup := rbac.GroupName
	return rbac.RoleRef{
		APIGroup: apiGroup,
		Kind:     ref.Kind,
		Name:     ref.Name,
	}
}

type RoleBindingCreateInput struct {
	NamespaceResourceCreateInput
	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects []rbac.Subject `json:"subjects,omitempty"`
	// RoleRef can reference a Role in the current namespace or a ClusterRole in the global namespace.
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	RoleRef *RoleRef `json:"role_ref"`
}

func (rb RoleBindingCreateInput) ToRoleBinding() *rbac.RoleBinding {
	return &rbac.RoleBinding{
		ObjectMeta: rb.ToObjectMeta(),
		Subjects:   rb.Subjects,
		RoleRef:    rb.RoleRef.ToRoleRef(),
	}
}
