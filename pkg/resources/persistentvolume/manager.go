package persistentvolume

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var PersistentVolumeManager *SPersistentVolumeManager

type SPersistentVolumeManager struct {
	*resources.SClusterResourceManager
}

func init() {
	PersistentVolumeManager = &SPersistentVolumeManager{
		SClusterResourceManager: resources.NewClusterResourceManager("persistentvolume", "persistentvolumes"),
	}
}
