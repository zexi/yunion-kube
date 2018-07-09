package client

import (
	"yunion.io/yunion-kube/pkg/types/apis"
)

type Node struct {
	RequestedHostname string             `json:"requestedHostname,omitempty"`
	ControlPlane      bool               `json:"controlPlane,omitempty"`
	Etcd              bool               `json:"etcd,omitempty"`
	Worker            bool               `json:"worker,omitempty"`
	CustomConfig      *apis.CustomConfig `json:"customConfig,omitempty"`
	DockerInfo        *apis.DockerInfo   `json:"dockerInfo,omitempty"`
}
