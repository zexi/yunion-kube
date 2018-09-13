package node

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var NodeManager *SNodeManager

type SNodeManager struct {
	*resources.SClusterResourceManager
}

func init() {
	NodeManager = &SNodeManager{
		SClusterResourceManager: resources.NewClusterResourceManager("k8s_node", "k8s_nodes"),
	}
}
