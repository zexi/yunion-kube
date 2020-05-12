package apis

type VirtualMachine struct {
	ObjectTypeMeta

	// Eip is elastic ip address
	Eip string `json:"eip"`
	// Hypervisor is virtual machine hypervisor
	Hypervisor string `json:"hypervisor"`
	// VcpuCount represents the number of CPUs of the virtual machine
	VcpuCount *int64 `json:"vcpuCount"`
	// VmemSizeGB reprensents the size of memory
	VmemSizeGB *int64 `json:"vmemSizeGB"`
	// InstanceType describes the specifications of the virtual machine
	InstanceType string `json:"instanceType"`

	VirtualMachineStatus
}

type OnecloudExternalInfoBase struct {
	// Id is resource cloud resource id
	Id string `json:"id"`
	// Status is resource cloud status
	Status string `json:"status"`
	// Action indicate the latest action for external vm
	Action string `json:"action"`
}

type VirtualMachineInfo struct {
	OnecloudExternalInfoBase
	// Ips is internal attached ip addresses
	Ips []string `json:"ips"`
}

type VirtualMachineStatus struct {
	Status       string `json:"status"`
	ExternalInfo VirtualMachineInfo `json:"externalInfo"`
	// CreateTimes record the continuous creation times
	CreateTimes int32 `json:"createTimes"`
}
