package pod

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var PodManager *SPodManager

type SPodManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	PodManager = &SPodManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("pod", "pods"),
	}
}
