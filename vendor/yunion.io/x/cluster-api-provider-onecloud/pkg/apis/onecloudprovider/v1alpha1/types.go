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

	// NetworkSpec encapsulates all things related to network.
	NetworkSpec NetworkSpec `json:"networkSpec,omitempty"`

	// The Yunion Region the cluster lives in.
	Region string `json:"region,omitempty"`

	// SSHKeyName is the name of the ssh key to attach to the bastion host
	SSHKeyName string `json:"sshKeyName,omitempty"`

	// CAKeyPair is the key pair for ca certs.
	CAKeyPair KeyPair `json:"caKeyPair,omitempty"`

	// EtcdCAKeyPair is the key pair for etcd.
	EtcdCAKeyPair KeyPair `json:"etcdCAKeyPair,omitempty"`

	// FrontProxyCAKeyPair is the key pair for FrontProxyKeyPair.
	FrontProxyCAKeyPair KeyPair `json:"frontProxyCAKeyPair,omitempty"`

	// SAKeyPair is the service account key pair.
	SAKeyPair KeyPair `json:"saKeyPair,omitempty"`
}

// NetworkSpec encapsulates all things related to network
type NetworkSpec struct {
	StaticLB *StaticLB `json:"staticLB,omitempty"`
}

// KeyPair is how operators can supply custom keypairs for kubeadm to use
type KeyPair struct {
	// base64 encoded cert and key
	Cert []byte `json:"cert"`
	Key  []byte `json:"key"`
}

// HasCertAndKey returns whether a keypair contains cert and key of non-zero length.
func (kp *KeyPair) HasCertAndKey() bool {
	return len(kp.Cert) != 0 && len(kp.Key) != 0
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneCloudClusterProviderStatus contains the status fields
// relevant to Yunion in the cluster object.
// +k8s:openapi-gen=true
type OneCloudClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Network Network `json:"network,omitempty"`
}

// Network encapsulates networking resources.
type Network struct {
	// APIServerELB is the Kubernetes api server classic load balancer.
	APIServerELB Loadbalancer
}

type StaticLB struct {
	// DNSName is the dns name of the load balancer.
	DNSName string `json:"dnsName,omitempty"`

	// IPAddress is the ip address of the load balancer.
	IPAddress string `json:"ipAddress,omitempty"`
}

type ClassicELB struct {
	// DNSName is the dns name of the load balancer.
	DNSName string `json:"dnsName,omitempty"`

	// IPAddress is the ip address of the load balancer.
	IPAddress string `json:"ipAddress,omitempty"`
}

type Loadbalancer struct {
	ClassicELB *ClassicELB `json:"classicELB,omitempty"`
	StaticLB   *StaticLB   `json:"staticLB,omitempty"`
}

func (lb Loadbalancer) GetDNSName() string {
	if lb.StaticLB != nil {
		if len(lb.StaticLB.DNSName) != 0 {
			return lb.StaticLB.DNSName
		}
		return lb.StaticLB.IPAddress
	}
	return lb.ClassicELB.DNSName
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
	MachineID    string `json:"machineID"`
	Role         string `json:"role"`
	PrivateIP    string `json:"privateIP,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneCloudMachineProviderStatus is the type that will be embedded in a Machine.Status.ProviderStatus field.
// It contains Yunion-specific status information.
// +k8s.openapi-gen=true
type OneCloudMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Machine is the machine created in Yunion
	// +optional
	Machine *Machine `json:"machine,omitempty"`

	// InstanceState is the state for this machine
	// +optional
	//InstanceState *string `json:"instanceState,omitempty"`

	// Conditions is a set of conditions associated with the Machine to indicate
	// errors or other status
	// +optional
	Conditions []OneCloudMachineProviderCondition `json:"conditions,omitempty"`
}

const (
	MachineStatusInit     = "init"
	MachineStatusCreating = "creating"
	MachineStatusPrepare  = "prepare"
	MachineStatusRunning  = "running"
	MachineStatusDeleting = "deleting"
)

var MachineDeployStatus []string = []string{
	MachineStatusPrepare,
	MachineStatusRunning,
}

const (
	RoleControlplane = "controlplane"
	RoleNode         = "node"
)

type Cluster struct {
	ID          string `json:"id"`
	ProjectId   string `json:"tenant_id"`
	Name        string `json:"name"`
	ClusterType string `json:"cluster_type"`
	CloudType   string `json:"cloud_type"`
	Mode        string `json:"mode"`
	Provider    string `json:"provider"`
}

// Machine describe an OneCloud guest or host
type Machine struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	ProjectId  string `json:"tenant_id"`
	Provider   string `json:"provider"`
	ClusterId  string `json:"cluster"`
	Role       string `json:"role"`
	ResourceID string `json:"resource_id"`
}

func (m Machine) NeedPrepare() bool {
	switch m.Status {
	case MachineStatusRunning, MachineStatusPrepare:
		return false
	}
	return true
}

func init() {
	SchemeBuilder.Register(&OneCloudClusterProviderSpec{})
	SchemeBuilder.Register(&OneCloudClusterProviderStatus{})
	SchemeBuilder.Register(&OneCloudMachineProviderSpec{})
	SchemeBuilder.Register(&OneCloudMachineProviderStatus{})
}
