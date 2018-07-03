package apis

type Cluster struct {
	Name   string
	Spec   ClusterSpec   `json:"spec"`
	Status ClusterStatus `json:"status"`
}

type ClusterSpec struct {
	ImportedConfig *ImportedConfig `json:"importedConfig,omitempty"`
}

type ImportedConfig struct {
	KubeConfig string `json:"kubeConfig"`
}

type ClusterStatus struct {
	ApiEndpoint string `json:"apiEndpoint,omitempty"`
}
