package configmap

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var ConfigMapManager *SConfigMapManager

type SConfigMapManager struct {
	*resources.SResourceBaseManager
}

func init() {
	ConfigMapManager = &SConfigMapManager{
		SResourceBaseManager: resources.NewResourceBaseManager("configmap", "configmaps"),
	}
}
