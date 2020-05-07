package apis


type ClusterResourceCreateInput struct {
	// 集群名称
	Cluster string `json:"cluster"`
}

type NamespaceResourceCreateInput struct {
	ClusterResourceCreateInput
	Namespace string `json:"namespace"`
}
