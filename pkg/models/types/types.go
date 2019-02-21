package types

// k8s cluster type
type ClusterType string

const (
	// common k8s cluster with nodes
	ClusterTypeDefault ClusterType = "default"
	// nodeless k8s cluster
	ClusterTypeServerless ClusterType = "serverless"
)

// k8s cluster cloud type
type CloudType string

const (
	// cluster running on private cloud
	CloudTypePrivate CloudType = "private"
	// cluster running on public cloud
	CloudTypePublic CloudType = "public"
	// cluster running on hybrid cloud
	CloudTypeHybrid CloudType = "hybrid"
)

type ModeType string

const (
	// self build k8s cluster
	ModeTypeSelfBuild ModeType = "customize"
	// public cloud managed k8s cluster
	ModeTypeManaged ModeType = "managed"
	// imported already exists k8s cluster
	ModeTypeImport ModeType = "import"
)

type ProviderType string

const (
	// system provider type by yunion YKE deploy
	ProviderTypeSystem ProviderType = "system"
	// default provider type by yunion onecloud
	ProviderTypeOnecloud ProviderType = "onecloud"
	// AWS provider
	ProviderTypeAws ProviderType = "aws"
	// Alibaba cloud provider
	ProviderTypeAliyun ProviderType = "aliyun"
	// Azure provider
	ProviderTypeAzure ProviderType = "azure"
	// Tencent cloud provider
	ProviderTypeQcloud ProviderType = "qcloud"
)

const (
	DefaultServiceCIDR   string = "10.43.0.0/16"
	DefaultServiceDomain string = "cluster.local"
)

type MachineResourceType string

const (
	MachineResourceTypeBaremetal = "baremetal"
	MachineResourceTypeVm        = "vm"
)

type RoleType string

const (
	RoleTypeControlplane = "controlplane"
	RoleTypeNode         = "node"
)

const (
	MachineStatusInit          = "init"
	MachineStatusCreating      = "creating"
	MachineStatusCreateFail    = "create_fail"
	MachineStatusPrepare       = "prepare"
	MachineStatusPrepareFail   = "prepare_fail"
	MachineStatusRunning       = "running"
	MachineStatusReady         = "ready"
	MachineStatusDeleting      = "deleting"
	MachineStatusDeleteFail    = "delete_fail"
	MachineStatusTerminating   = "terminating"
	MachineStatusTerminateFail = "terminate_fail"

	ClusterStatusInit       = "init"
	ClusterStatusCreating   = "creating"
	ClusterStatusCreateFail = "create_fail"
	ClusterStatusRunning    = "running"
	ClusterStatusUnknown    = "unknown"
	ClusterStatusDeleting   = "deleting"
	ClusterStatusDeleteFail = "delete_fail"
)

type CreateClusterData struct {
	Name          string               `json:"name"`
	Namespace     string               `json:"namespace"`
	ClusterType   string               `json:"cluster_type"`
	CloudType     string               `json:"cloud_type"`
	Mode          string               `json:"mode"`
	Provider      string               `json:"provider"`
	ServiceCidr   string               `json:"service_cidr"`
	ServiceDomain string               `json:"service_domain"`
	PodCidr       string               `json:"pod_cidr"`
	Version       string               `json:"version"`
	HA            bool                 `json:"ha"`
	Machines      []*CreateMachineData `json:"machines"`
}

type CreateMachineData struct {
	Name         string `json:"name"`
	ClusterId    string `json:"cluster_id"`
	Role         string `json:"role"`
	ResourceType string `json:"resource_type"`
	ResourceId   string `json:"resource_id"`
}

type Machine struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	ClusterId  string `json:"cluster_id"`
	Role       string `json:"role"`
	InstanceId string `json:"instance_id"`
}

type KeyPair struct {
	// base64 encoded cert and key
	Cert []byte `json:"cert"`
	Key  []byte `json:"key"`
}

func (kp KeyPair) HasCertAndKey() bool {
	return len(kp.Cert) != 0 && len(kp.Key) != 0
}

const (
	ContainerSchedtag = "container"
	DefaultCluster    = "default"
)
