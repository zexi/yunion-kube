package apis

import (
	"yunion.io/yke/pkg/types"
)

type Cluster struct {
	Id                           string
	Name                         string
	ApiEndpoint                  string `json:"apiEndpoint"`
	CaCert                       string `json:"caCert"`
	YunionKubernetesEngineConfig *types.KubernetesEngineConfig
}
