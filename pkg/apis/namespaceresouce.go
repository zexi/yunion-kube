package apis

type NamespaceResourceCreateInput struct {
	ClusterResourceCreateInput
	// 命名空间
	Namespace string `json:"namespace"`
}

type NamespaceResourceListInput struct {
	ClusterResourceListInput
	// 命名空间
	Namespace string `json:"namespace"`
}

type NamespaceResourceDetail struct {
	ClusterResourceDetail

	NamespaceId string `json:"namespace_id"`
	Namespace   string `json:"namespace"`
}
