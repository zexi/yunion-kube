package api

import (
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FederatedClusterRoleBindingCreateInput struct {
	FederatedResourceCreateInput
	Spec *FederatedClusterRoleBindingSpec `json:"spec"`
}

type FederatedClusterRoleBindingSpec struct {
	Template ClusterRoleBindingTemplate `json:"template"`
}

func (spec FederatedClusterRoleBindingSpec) ToClusterRoleBinding(objMeta metav1.ObjectMeta) *rbac.ClusterRoleBinding {
	return &rbac.ClusterRoleBinding{
		ObjectMeta: objMeta,
		Subjects:   spec.Template.Subjects,
	}
}

type ClusterRoleBindingTemplate struct {
	Subjects Subjects `json:"subjects,omitempty"`
	// RoleRef can only reference a FederatedClusterRole in the global namespace.
	RoleRef RoleRef `json:"roleRef"`
}
