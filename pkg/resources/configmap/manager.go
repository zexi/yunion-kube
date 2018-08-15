package configmap

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var ConfigMapManager *SConfigMapManager

type SConfigMapManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	ConfigMapManager = &SConfigMapManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("configmap", "configmaps"),
	}
}
