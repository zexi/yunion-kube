package apis

type K8sClusterResourceGetInput struct {
	// required: true
	Cluster string `json:"cluster"`
}

type K8sNamespaceResourceGetInput struct {
	K8sClusterResourceGetInput
	// required: true
	Namespace string `json:"namespace"`
}

type K8sClusterResourceCreateInput struct {
	// required: true
	Cluster string `json:"cluster"`
	// required: true
	Name string `json:"name"`
}

type K8sNamespaceResourceCreateInput struct {
	K8sClusterResourceCreateInput
	// required: true
	Namespace string `json:"namespace"`
}
