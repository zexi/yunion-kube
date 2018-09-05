package node

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var NodeManager *SNodeManager

type SNodeManager struct {
	*resources.SResourceBaseManager
}

func init() {
	NodeManager = &SNodeManager{
		SResourceBaseManager: resources.NewResourceBaseManager("k8s_node", "k8s_nodes"),
	}
}
