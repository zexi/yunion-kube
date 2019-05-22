package apis

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type MachineCreateConfig struct {
	Vm *MachineCreateVMConfig `json:"vm,omitempty"`
}

type MachineCreateVMConfig struct {
	PreferRegion     string `json:"prefer_region_id"`
	PreferZone       string `json:"prefer_zone_id"`
	PreferWire       string `json:"prefer_wire_id"`
	PreferHost       string `json:"prefer_host_id"`
	PreferBackupHost string `json:"prefer_backup_host"`

	Disks           []*api.DiskConfig           `json:"disks"`
	Networks        []*api.NetworkConfig        `json:"nets"`
	Schedtags       []*api.SchedtagConfig       `json:"schedtags"`
	IsolatedDevices []*api.IsolatedDeviceConfig `json:"isolated_devices"`

	Hypervisor string `json:"hypervisor"`
	VmemSize   int    `json:"vmem_size"`
	VcpuCount  int    `json:"vcpu_count"`
}

type MachinePrepareInput struct {
	FirstNode bool   `json:"first_node"`
	Role      string `json:"role"`

	CAKeyPair           *KeyPair `json:"ca_key_pair"`
	EtcdCAKeyPair       *KeyPair `json:"etcd_ca_key_pair"`
	FrontProxyCAKeyPair *KeyPair `json:"front_proxy_ca_key_pair"`
	SAKeyPair           *KeyPair `json:"sa_key_pair"`
	BootstrapToken      string   `json:"bootstrap_token"`
	ELBAddress          string   `json:"elb_address"`

	Config *MachineCreateConfig `json:"config"`

	InstanceId string `json:"-"`
	PrivateIP  string `json:"-"`
}

const (
	DefaultVMMemSize      = 2048       // 2G
	DefaultVMCPUCount     = 2          // 2 core
	DefaultVMRootDiskSize = 100 * 1024 // 100G
)

const (
	MachineMetadataCreateParams = "create_params"
)
