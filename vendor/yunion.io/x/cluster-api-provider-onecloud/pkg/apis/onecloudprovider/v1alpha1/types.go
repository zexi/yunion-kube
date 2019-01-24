/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OneCloudMachineProviderConditionType string

// OneCloudMachineProviderCondition is a condition in a OnecloudMachineProviderStatus
type OneCloudMachineProviderCondition struct {
	// Type is the type of the condition.
	Type OneCloudMachineProviderConditionType `json:"type"`
	// Status is the status of the condition.
	Status corev1.ConditionStatus `json:"status"`
	// LastProbeTime is the last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime"`
	// LastTransitionTime is the last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Reason is a unique, one-word, CamelCase reaons for the condition's last transition.
	// +optional
	Reason string `json:"reason"`
	// Message is a human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneCloudClusterProviderSpec is the providerConfig for Yunion in the cluster
// object
// +k8s:openapi-gen=true
type OneCloudClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The Yunion Region the cluster lives in.
	Region string `json:"region,omitempty"`

	// CACertificate is a PEM encoded CA Certificate for the control plane nodes.
	CACertificate []byte `json:"caCertificate,omitempty"`

	// CAPrivateKey is a PEM encoded PKCS1 CA PrivateKey for the control plane nodes.
	CAPrivateKey []byte `json:"caKey,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneCloudClusterProviderStatus contains the status fields
// relevant to Yunion in the cluster object.
// +k8s:openapi-gen=true
type OneCloudClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneCloudMachineProviderSpec is the type that will be embedded in a Machine.Spec.ProviderSpec field
// for an OneCloud host instance. It is used by machine actuator to create a host instance,
// +k8s:openapi-gen=true
type OneCloudMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ResourceType determine machine platform to run. Example: baremetal or vm
	ResourceType string `json:"resourceType"`
	Provider     string `json:"provider"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneCloudMachineProviderStatus is the type that will be embedded in a Machine.Status.ProviderStatus field.
// It contains Yunion-specific status information.
// +k8s.openapi-gen=true
type OneCloudMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// InstanceID is the instance ID of the machine created in Yunion
	// +optional
	InstanceID *string `json:"instanceID,omitempty"`

	// InstanceState is the state for this machine
	// +optional
	InstanceState *string `json:"instanceState,omitempty"`

	// Conditions is a set of conditions associated with the Machine to indicate
	// errors or other status
	// +optional
	Conditions []OneCloudMachineProviderCondition `json:"conditions,omitempty"`
}

// Instance describe an OneCloud guest or host
type Instance struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	ProjectId string `json:"project_id"`
	Provider  string `json:"provider"`
	Cluster   string `json:"cluster"`
	Role      string `json:"role"`
	Type      string `json:"type"`
}

func init() {
	SchemeBuilder.Register(&OneCloudClusterProviderSpec{})
	SchemeBuilder.Register(&OneCloudClusterProviderStatus{})
	SchemeBuilder.Register(&OneCloudMachineProviderSpec{})
	SchemeBuilder.Register(&OneCloudMachineProviderStatus{})
}
