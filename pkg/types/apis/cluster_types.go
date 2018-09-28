package apis

import (
	"yunion.io/x/yke/pkg/types"
)

type Cluster struct {
	Id                           string
	Name                         string
	ApiEndpoint                  string `json:"apiEndpoint"`
	CaCert                       string `json:"caCert"`
	YunionKubernetesEngineConfig *types.KubernetesEngineConfig
}
