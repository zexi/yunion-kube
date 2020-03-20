package apis

import "yunion.io/x/onecloud/pkg/apis"

const (
	ClusterComponentCephCSI = "cephCSI"
)

type ComponentCreateInput struct {
	apis.Meta

	Name string `json:"name"`
	Type string `json:"type"`

	ComponentSettings
}

type ComponentSettings struct {
	Namespace string                   `json:"namespace"`
	CephCSI   *ComponentSettingCephCSI `json:"cephCSI"`
}

type ComponentCephCSIConfigCluster struct {
	ClsuterId string   `json:"clusterId"`
	Monitors  []string `json:"monitors"`
}

type ComponentSettingCephCSI struct {
	Config []ComponentCephCSIConfigCluster `json:"config"`
}

type ComponentsStatus struct {
	apis.Meta

	CephCSI *ComponentStatusCephCSI `json:"cephCSI"`
}

type ComponentStatus struct {
	Id      string `json:"id"`
	Created bool   `json:"created"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

type ComponentStatusCephCSI struct {
	ComponentStatus
}

type ComponentUpdateInput struct {
	apis.Meta

	Type string `json:"type"`

	ComponentSettings
}
