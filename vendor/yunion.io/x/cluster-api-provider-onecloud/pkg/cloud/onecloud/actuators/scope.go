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

package actuators

import (
	"github.com/pkg/errors"
	"k8s.io/klog"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"

	//"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"
)

// ScopeParams defines the input parameters used to create a new Scope
type ScopeParams struct {
	Cluster *clusterv1.Cluster
	Client  client.ClusterV1alpha1Interface
}

// NewScope creates a new Scope from the supplied parameters.
// This is meant to be called for each different actuator iteration.
func NewScope(params ScopeParams) (*Scope, error) {
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil cluster")
	}

	clusterConfig, err := v1alpha1.ClusterConfigFromProviderSpec(params.Cluster.Spec.ProviderSpec)
	if err != nil {
		return nil, errors.Errorf("failed to load cluster provider config: %v", err)
	}

	clusterStatus, err := v1alpha1.ClusterStatusFromProviderStatus(params.Cluster.Status.ProviderStatus)
	if err != nil {
		return nil, errors.Errorf("failed to load cluster provider status: %v", err)
	}

	var clusterClient client.ClusterInterface
	if params.Client != nil {
		clusterClient = params.Client.Clusters(params.Cluster.Namespace)
	}

	return &Scope{
		Cluster:       params.Cluster,
		ClusterClient: clusterClient,
		ClusterConfig: clusterConfig,
		ClusterStatus: clusterStatus,
	}, nil
}

// Scope defines the basic context for an actuator to operate upon.
type Scope struct {
	YunionClients
	Cluster       *clusterv1.Cluster
	ClusterClient client.ClusterInterface
	ClusterConfig *v1alpha1.OneCloudClusterProviderSpec
	ClusterStatus *v1alpha1.OneCloudClusterProviderStatus
}

// Name returns the cluster name
func (s *Scope) Name() string {
	return s.Cluster.Name
}

// Namespace returns the cluster namespace
func (s *Scope) Namespace() string {
	return s.Cluster.Namespace
}

// Region returns the cluster region
func (s *Scope) Region() string {
	return s.ClusterConfig.Region
}

// Network returns the cluster network object.
func (s *Scope) Network() *v1alpha1.Network {
	return &s.ClusterStatus.Network
}

func (s *Scope) storeClusterConfig(cluster *clusterv1.Cluster) (*clusterv1.Cluster, error) {
	ext, err := v1alpha1.EncodeClusterSpec(s.ClusterConfig)
	if err != nil {
		return nil, err
	}

	cluster.Spec.ProviderSpec.Value = ext
	return s.ClusterClient.Update(cluster)
}

func (s *Scope) storeClusterStatus(cluster *clusterv1.Cluster) (*clusterv1.Cluster, error) {
	ext, err := v1alpha1.EncodeClusterStatus(s.ClusterStatus)
	if err != nil {
		return nil, err
	}

	cluster.Status.ProviderStatus = ext
	return s.ClusterClient.UpdateStatus(cluster)
}

func (s *Scope) Close() {
	if s.ClusterClient == nil {
		return
	}

	latestCluster, err := s.storeClusterConfig(s.Cluster)
	if err != nil {
		klog.Errorf("[scope] failed to store provider config for cluster %q in namespace %q: %v", s.Cluster.Name, s.Cluster.Namespace, err)
	}

	_, err = s.storeClusterStatus(latestCluster)
	if err != nil {
		klog.Errorf("[scope] failed to store provider status for cluster %q in namespace %q: %v", s.Cluster.Name, s.Cluster.Namespace, err)
	}
}

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	Cluster *clusterv1.Cluster
	Machine *clusterv1.Machine
	Client  client.ClusterV1alpha1Interface
}

// NewMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each machine actuator operation.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	scope, err := NewScope(ScopeParams{Client: params.Client, Cluster: params.Cluster})
	if err != nil {
		return nil, err
	}

	machineConfig, err := v1alpha1.MachineConfigFromProviderSpec(params.Machine.Spec.ProviderSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get machine config")
	}

	machineStatus, err := v1alpha1.MachineStatusFromProviderStatus(params.Machine.Status.ProviderStatus)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get machine provider status")
	}

	var machineClient client.MachineInterface
	if params.Client != nil {
		machineClient = params.Client.Machines(params.Machine.Namespace)
	}

	return &MachineScope{
		Scope:         scope,
		Machine:       params.Machine,
		MachineClient: machineClient,
		MachineConfig: machineConfig,
		MachineStatus: machineStatus,
	}, nil
}

type MachineScope struct {
	*Scope

	Machine       *clusterv1.Machine
	MachineClient client.MachineInterface
	MachineConfig *v1alpha1.OneCloudMachineProviderSpec
	MachineStatus *v1alpha1.OneCloudMachineProviderStatus
}

func (m *MachineScope) Name() string {
	return m.Machine.Name
}

func (m *MachineScope) Namespace() string {
	return m.Machine.Namespace
}

// Role returns the machine role from the labels.
func (m *MachineScope) Role() string {
	//return m.Machine.Labels["set"]
	return m.MachineConfig.Role
}

// Region returns the machine region.
func (m *MachineScope) Region() string {
	return m.Scope.Region()
}

func (m *MachineScope) storeMachineSpec(machine *clusterv1.Machine) (*clusterv1.Machine, error) {
	ext, err := v1alpha1.EncodeMachineSpec(m.MachineConfig)
	if err != nil {
		return nil, err
	}

	machine.Spec.ProviderSpec.Value = ext
	return m.MachineClient.Update(machine)
}

func (m *MachineScope) storeMachineStatus(machine *clusterv1.Machine) (*clusterv1.Machine, error) {
	ext, err := v1alpha1.EncodeMachineStatus(m.MachineStatus)
	if err != nil {
		return nil, err
	}

	machine.Status.ProviderStatus = ext
	return m.MachineClient.UpdateStatus(machine)
}

func (m *MachineScope) Close() {
	if m.MachineClient == nil {
		return
	}

	latestMachine, err := m.storeMachineSpec(m.Machine)
	if err != nil {
		klog.Errorf("[machinescope] failed to update machine %q in namespace %q: %v", m.Machine.Name, m.Machine.Namespace, err)
	}

	_, err = m.storeMachineStatus(latestMachine)
	if err != nil {
		klog.Errorf("[machinescope] failed to store provider status for machine %q in namespace %q: %v", m.Machine.Name, m.Machine.Namespace, err)
	}
}
