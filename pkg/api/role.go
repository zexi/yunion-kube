package api

import (
	rbac "k8s.io/api/rbac/v1"
)

type RoleCreateInput struct {
	NamespaceResourceCreateInput
	Rules []rbac.PolicyRule `json:"rules"`
}

func (input RoleCreateInput) ToRole() *rbac.Role {
	objMeta := input.NamespaceResourceCreateInput.ToObjectMeta()
	return &rbac.Role{
		ObjectMeta: objMeta,
		Rules:      input.Rules,
	}
}
