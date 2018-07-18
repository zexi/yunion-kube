package apis

import (
	"yunion.io/yke/pkg/types"
)

type Node struct {
	ClusterId         string            `json:"clusterId"`
	Name              string            `json:"name"`
	Etcd              bool              `json:"etcd"`
	ControlPlane      bool              `json:"controlPlane"`
	Worker            bool              `json:"worker"`
	RequestedHostname string            `json:"requestedHostname"`
	Address           string            `json:"address"`
	CustomConfig      *CustomConfig     `json:"customConfig"`
	DockerInfo        *DockerInfo       `json:"dockerInfo"`
	NodeConfig        *types.ConfigNode `json:"ykeNodeConfig"`
}

type CustomConfig struct {
	Address         string   `json:"address"`
	InternalAddress string   `json:"internalAddress"`
	DockerSocket    string   `json:"dockerSocket"`
	Roles           []string `json:"roles"`
}
