package storageclass

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var StorageClassManager *SStorageClassManager

type SStorageClassManager struct {
	*resources.SClusterResourceManager
}

func init() {
	StorageClassManager = &SStorageClassManager{
		SClusterResourceManager: resources.NewClusterResourceManager("storageclass", "storageclasses"),
	}
}
