package apis

import (
	"yunion.io/yke/pkg/types"
)

type Cluster struct {
	Id                           string
	Name                         string
	Spec                         ClusterSpec   `json:"spec"`
	Status                       ClusterStatus `json:"status"`
	ApiEndpoint                  string        `json:"apiEndpoint"`
	CaCert                       string        `json:"caCert"`
	YunionKubernetesEngineConfig *types.KubernetesEngineConfig
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
