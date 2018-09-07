package persistentvolume

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var PersistentVolumeManager *SPersistentVolumeManager

type SPersistentVolumeManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	PersistentVolumeManager = &SPersistentVolumeManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("persistentvolume", "persistentvolumes"),
	}
}
