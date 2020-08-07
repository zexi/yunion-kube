package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceResourceCreateInput struct {
	ClusterResourceCreateInput

	// required: true
	// 命名空间
	NamespaceId string `json:"namespace_id"`
	// Namespace should set by backend
	// swagger:ignore
	Namespace string `json:"namespace" yunion-deprecated-by:"namespace_id"`
}

func (input NamespaceResourceCreateInput) ToObjectMeta() metav1.ObjectMeta {
	objMeta := input.ClusterResourceCreateInput.ToObjectMeta()
	objMeta.Namespace = input.Namespace
	return objMeta
}

type NamespaceResourceListInput struct {
	ClusterResourceListInput
	// 命名空间
	Namespace string `json:"namespace"`
}

type NamespaceResourceDetail struct {
	ClusterResourceDetail

	NamespaceId string `json:"namespace_id"`
	Namespace   string `json:"namespace"`
}
