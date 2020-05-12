package apis

import "yunion.io/x/onecloud/pkg/apis"

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
	// system provider type means default v3 supervisor cluster
	ProviderTypeSystem ProviderType = "system"
	// default provider type by onecloud
	ProviderTypeOnecloud ProviderType = "onecloud"
	// AWS provider
	ProviderTypeAws ProviderType = "aws"
	// Alibaba cloud provider
	ProviderTypeAliyun ProviderType = "aliyun"
	// Azure provider
	ProviderTypeAzure ProviderType = "azure"
	// Tencent cloud provider
	ProviderTypeQcloud ProviderType = "qcloud"
	// External provider type by import
	ProviderTypeExternal ProviderType = "external"
)

const (
	DefaultServiceCIDR   string = "10.43.0.0/16"
	DefaultServiceDomain string = "cluster.local"
	DefaultPodCIDR       string = "10.42.0.0/16"
)

type ClusterResourceType string

const (
	ClusterResourceTypeHost    = "host"
	ClusterResourceTypeGuest   = "guest"
	ClusterResourceTypeUnknown = "unknown"
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

	ClusterStatusInit              = "init"
	ClusterStatusCreating          = "creating"
	ClusterStatusCreateFail        = "create_fail"
	ClusterStatusCreatingMachine   = "creating_machine"
	ClusterStatusCreateMachineFail = "create_machine_fail"
	ClusterStatusRunning           = "running"
	ClusterStatusLost              = "lost"
	ClusterStatusUnknown           = "unknown"
	ClusterStatusError             = "error"
	ClusterStatusDeleting          = "deleting"
	ClusterStatusDeleteFail        = "delete_fail"
)

type ClusterCreateInput struct {
	apis.Meta

	Name            string               `json:"name"`
	ClusterType     string               `json:"cluster_type"`
	CloudType       string               `json:"cloud_type"`
	Mode            string               `json:"mode"`
	Provider        string               `json:"provider"`
	ServiceCidr     string               `json:"service_cidr"`
	ServiceDomain   string               `json:"service_domain"`
	PodCidr         string               `json:"pod_cidr"`
	Version         string               `json:"version"`
	HA              bool                 `json:"ha"`
	Machines        []*CreateMachineData `json:"machines"`
	ImageRepository *ImageRepository     `json:"image_repository"`

	// imported cluster data
	ImportClusterData
}

type ImageRepository struct {
	// url define cluster image repository url, e.g: registry.hub.docker.com/yunion
	Url string `json:"url"`
	// if insecure, the /etc/docker/daemon.json insecure-registries will add this registry
	Insecure bool `json:"insecure"`
}

type CreateMachineData struct {
	Name         string               `json:"name"`
	ClusterId    string               `json:"cluster_id"`
	Role         string               `json:"role"`
	Provider     string               `json:"provider"`
	ResourceType string               `json:"resource_type"`
	ResourceId   string               `json:"resource_id"`
	Address      string               `json:"address"`
	FirstNode    bool                 `json:"first_node"`
	Config       *MachineCreateConfig `json:"config"`
}

type ImportClusterData struct {
	Kubeconfig string `json:"kubeconfig"`
	ApiServer  string `json:"api_server"`
}

const (
	ContainerSchedtag = "container"
	DefaultCluster    = "default"
)

type UsableInstance struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type ClusterListInput struct {
	apis.SharableVirtualResourceListInput
}
