package apis

type ClusterCreateInput struct {
	Name          string               `json:"name"`
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
	// imported cluster data
	ImportClusterData
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

/// Cluster define kubernetes cluster instance
type SCluster struct {
	SSharableVirtualResourceBase

	ClusterType   string `json:"cluster_type"`
	CloudType     string `json:"cloud_type"`
	ResourceType  string `json:"resource_type"`
	Mode          string `json:"mode"`
	Provider      string `json:"provider"`
	ServiceCidr   string `json:"service_cidr"`
	ServiceDomain string `json:"service_domain"`
	PodCidr       string `json:"pod_cidr"`
	Version       string `json:"version"`
	Ha            bool   `json:"ha"`

	// kubernetes config
	Kubeconfig string `json:"kubeconfig"`

	// kubernetes api server endpoint
	ApiServer string `json:"api_server"`
}

type ClusterCreateOutput struct {
	SCluster
}
