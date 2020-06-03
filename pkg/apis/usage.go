package apis

type GlobalUsage struct {
	AllUsage     *UsageResult `json:"all"`
	DomainUsage  *UsageResult `json:"domain"`
	ProjectUsage *UsageResult `json:"project"`
}

type UsageResult struct {
	ClusterUsage *ClusterUsage `json:"cluster"`
}

type MemoryUsage struct {
	Capacity int64 `json:"capacity"`
	Request  int64 `json:"request"`
	Limit    int64 `json:"limit"`
}

func (u *MemoryUsage) Add(ou *MemoryUsage) *MemoryUsage {
	u.Capacity += ou.Capacity
	u.Request += ou.Request
	u.Limit += ou.Limit
	return u
}

type CpuUsage struct {
	Capacity int64 `json:"capacity"`
	Request  int64 `json:"request"`
	Limit    int64 `json:"limit"`
}

func (u *CpuUsage) Add(ou *CpuUsage) *CpuUsage {
	u.Capacity += ou.Capacity
	u.Request += ou.Request
	u.Limit += ou.Limit
	return u
}

type PodUsage struct {
	Capacity int64 `json:"capacity"`
	Count    int64 `json:"count"`
}

func (u *PodUsage) Add(ou *PodUsage) *PodUsage {
	u.Capacity += ou.Capacity
	u.Count += ou.Count
	return u
}

type NodeUsage struct {
	Memory        *MemoryUsage `json:"memory"`
	Cpu           *CpuUsage    `json:"cpu"`
	Pod           *PodUsage    `json:"pod"`
	Count         int64        `json:"count"`
	ReadyCount    int64        `json:"ready_count"`
	NotReadyCount int64        `json:"not_ready_count"`
}

func (u *NodeUsage) Add(ou *NodeUsage) *NodeUsage {
	u.Memory.Add(ou.Memory)
	u.Cpu.Add(ou.Cpu)
	u.Pod.Add(ou.Pod)
	u.Count += ou.Count
	u.ReadyCount += ou.ReadyCount
	u.NotReadyCount += ou.NotReadyCount
	return u
}

func NewNodeUsage() *NodeUsage {
	return &NodeUsage{
		Memory: new(MemoryUsage),
		Cpu:    new(CpuUsage),
		Pod:    new(PodUsage),
	}
}

type ClusterUsage struct {
	Node  *NodeUsage `json:"node"`
	Count int64      `json:"count"`
}
