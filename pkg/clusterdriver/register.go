package clusterdriver

import (
	"yunion.io/x/yunion-kube/pkg/clusterdriver/types"
	"yunion.io/x/yunion-kube/pkg/clusterdriver/yke"
)

var Drivers map[string]types.Driver

func init() {
	Drivers = map[string]types.Driver{
		"yke": yke.NewDriver(),
	}
}
